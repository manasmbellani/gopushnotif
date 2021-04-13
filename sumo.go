package main

import (
	"log"
	"os"

	"github.com/go-resty/resty/v2"
)

// getSumoCollectorURL is used to retrieve the Pushover user key from AWS secret or the 
// Environment variable
func getSumoCollectorURL(collectorURL string, collectorURLAWSSecret string, awsRegion string, 
	awsProfile string, pullSecretsFromAWS bool, sendToSumo bool) string {
	
	if sendToSumo {
		if collectorURL == "" {
			if pullSecretsFromAWS {
				// Pull the secret from AWS Secrets Manager
				collectorURL = GetAWSSecret(collectorURLAWSSecret, awsRegion, awsProfile)	
			} else {
				// Check if Pushover User Key supplied in env vars
				collectorURL = os.Getenv(SumoCollectorURL)
				if collectorURL == "" {
					log.Fatalf("[-] Collector URL must be specified either as input OR in env var")
				}
			}
		}
	}
	
	return collectorURL
}



// configureResty is used to configure the resty client e.g. the user agent 
// string and other global client settings are set here
func configureResty(restyClient *resty.Client, userAgentString string) {
	restyClient.SetHeaders(
		map[string]string {
			"Contenty-Type": "application/json",
			"User-Agent": userAgentString,
		},
	)
}

// sendMessageViaSumo sends a custom message to the specified sumo collectorURL
func sendMessageViaSumo(restyClient *resty.Client, collectorURL string, 
	message string) {
	
	resp, err := restyClient.R().SetBody(message).Post(collectorURL)
	if err != nil {
		log.Printf("[*] Error sending message to Sumo. Err: %s\n", err.Error())
	} else {
		statusCode := resp.StatusCode()
		log.Printf("[+] Message sent to Sumo. Status code: %d\n", statusCode)
	}
	
}