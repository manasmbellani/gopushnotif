package main

import (
	"log"
	"os"
	"bufio"

	"github.com/gregdel/pushover"
)

// getUserKey is used to retrieve the Pushover user key from AWS secret or the 
// Environment variable
func getUserKey(userKey string, pushoverUserKeyAWSSecret string, awsRegion string, 
	awsProfile string, pullSecretsFromAWS bool, sendToPushover bool) string {
	
	if sendToPushover {
		if userKey == "" {
			if pullSecretsFromAWS {
				// Pull the secret from AWS Secrets Manager
				userKey = GetAWSSecret(pushoverUserKeyAWSSecret, awsRegion, awsProfile)	
			} else {
				// Check if Pushover User Key supplied in env vars
				userKey = os.Getenv(PushoverUserKey)
				if userKey == "" {
					log.Fatalf("[-] Pushover User Key must be specified either as input OR in env var")
				}
			}
		}
	}
	
	return userKey
}


// getAppToken is used to retrieve the Pushover app token from environment var
// or the AWS Secret
func getAppToken(appToken string, pushoverAppTokenAWSSecret string, awsRegion string, 
	awsProfile string, pullSecretsFromAWS bool, sendToPushover bool) string {
	
	if sendToPushover {
		if appToken == "" {
			if pullSecretsFromAWS {
				// Pull the secret from AWS Secrets Manager
				appToken = GetAWSSecret(pushoverAppTokenAWSSecret, awsRegion, awsProfile)
			} else {
				// Check if Pushover App Token supplied in env vars
				appToken = os.Getenv(PushoverAppToken)
				if appToken == "" {
					log.Fatalf("[-] Pushover App Token must be specified either as input OR in env var")
				}
			}
		}
	}
	return appToken
}

// sendMessageViaPushover sent the message via pushover
func sendMessageViaPushover(app *pushover.Pushover, recipient *pushover.Recipient, 
	line string, attachment string, outfile string, dryRun bool) {

	if !dryRun {
		// Create the message to send
		message := pushover.NewMessage(line)
		if attachment != "" {
			file, _ := os.Open(attachment)
			message.AddAttachment(bufio.NewReader(file))
		}

		// Attach the screenshot output file if taken successfully
		if outfile != "" {

			// First check if it even exists - sometimes due to err
			// screenshot may not be taken
			_, err := os.Stat(outfile)
			if !os.IsNotExist(err) {
				file, _ := os.Open(outfile)
				message.AddAttachment(bufio.NewReader(file))
			} else {
				log.Printf("[!] File: %s did not exist. Can't send screenshot.", outfile)
			}
		}

		// Send the message with optional screenshot, and response
		// details too
		response, err := app.SendMessage(message, recipient)
		if err != nil {
			log.Println(err)
		}

		// Print the Pushover API response
		log.Printf("[*] Pushover API Response: %+v", response)
	}
}