// Function to send go push notification to pushover app.
// It also recognises special lines in the format [sig] urls if `-p` flag is set
// and takes a screenshot via gowitness (which should be installed in PATH dir)
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gregdel/pushover"
	"github.com/go-resty/resty/v2"
)

// ScreenshotFolderPrefix - Folder to store the screenshot
const ScreenshotFolderPrefix = "out-screenshots"

// ScreenshotFilePrefix - Prefix for the screenshot
const ScreenshotFilePrefix = "out-screenshot"

// PushoverUserKey - Pushover user key name in env var
const PushoverUserKey = "PUSHOVER_USER_KEY"

// PushoverAppToken - Pushover app token name in env var
const PushoverAppToken = "PUSHOVER_APP_TOKEN"

// SumologicCollectorURL - Sumologic collector URL name in env var
const SumoCollectorURL = "SUMOLOGIC_COLLECTOR_URL"

// UserAgentString - is the default user agent string to use for sending 
// messages to Sumo
const UserAgentString = "Mozilla/5.0 (Macintosh; Intel Mac OS X 11_2_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36"

func getRegexGroups(regEx, url string) (paramsMap map[string]string) {

	var compRegEx = regexp.MustCompile(regEx)
	match := compRegEx.FindStringSubmatch(url)

	paramsMap = make(map[string]string)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}
	return paramsMap
}

// Determine if line mentioned is a url with HTTP/HTTPS protocol supplied
func isURLWithHTTPProtocol(line string) bool {
	if strings.Index(line, "http://") != -1 ||
		strings.Index(line, "https://") != -1 {
		return true
	}
	return false
}

// Execute a command to get the output, error. Command is executed when in the
// optionally specified 'cmdDir' OR it is executed with the current working dir
func execCmd(cmdToExec string, cmdDir string, dryRun bool) string {
	// Get the original working directory
	owd, _ := os.Getwd()

	// Switch to the directory
	if cmdDir != "" {
		os.Chdir(cmdDir)
	}

	// Get my current working directory
	cwd, _ := os.Getwd()

	log.Printf("[v] Executing cmd: %s in dir: %s\n", cmdToExec, cwd)

	totalOut := ""
	if !dryRun {

		// Determine the command to execute based on OS
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("cmd.exe", "/c", cmdToExec)
		default:
			cmd = exec.Command("/bin/sh", "-c", cmdToExec)
		}

		// Execute the command
		out, err := cmd.CombinedOutput()
		var outStr, errStr string
		if out == nil {
			outStr = ""
		} else {
			outStr = string(out)
		}

		if err == nil {
			errStr = ""
		} else {
			errStr = string(err.Error())
			//log.Printf("Command Error: %s\n", err)
		}

		totalOut = (outStr + "\n" + errStr)
	}

	// Switch back to the original working directory
	os.Chdir(owd)

	return totalOut
}

// Take screenshot utilising 'gowitness' and return screenshot output file path
func takeScreenshot(url string, screenshotFolder string, screenshotName string,
	gowitnessBin string, screenshotRes string, timeout int, dryRun bool) string {

	// Output file path where screenshot is written
	outfile := ""

	// Default to screenshot.png if not specified
	if screenshotName == "" {
		screenshotName = "screenshot.png"
	}

	// Prepare screenshot template by clearing screenshots from the folder and
	// then take gowitness in the folder
	screenshotTmp := "rm {screenshots_folder}/*; "
	screenshotTmp += "{gowitness} single --url {url} -d {screenshots_folder} --chrome-timeout {timeout} -R {screenshot_res};"
	screenshotTmp += "mv {screenshots_folder}/*.png {screenshots_folder}/{screenshot_name}"

	// Create the screenshot folder if it doesn't exist
	log.Printf("Creating folder: %s\n", screenshotFolder)
	if !dryRun {
		_, err := os.Stat(screenshotFolder)

		if os.IsNotExist(err) {
			os.Mkdir(screenshotFolder, 0755)
		}
	}

	if isURLWithHTTPProtocol(url) {
		// Take the screenshot
		screenshotCmd := screenshotTmp
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{screenshots_folder}",
			screenshotFolder)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{screenshot_res}", screenshotRes)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{url}", url)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{screenshot_name}",
			screenshotName)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{gowitness}",
			gowitnessBin)

		timeoutStr := strconv.FormatInt(int64(timeout), 10)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{timeout}",
			timeoutStr)
		outfile = path.Join(screenshotFolder, screenshotName)

		execCmd(screenshotCmd, "", dryRun)
	}

	return outfile

}

func main() {
	var dryRun bool
	var userKey string
	var appToken string
	var attachment string
	var timeout int
	var parseSignature bool
	var verbose bool
	var gowitness string
	var numThreads int
	var screenshotRes string
	var sendUnique bool
	var collectorURL string	
	var sendToPushover bool
	var sendToSumo bool
	var pullSecretsFromAWS bool
	var collectorURLAWSSecret string
	var pushoverUserKeyAWSSecret string
	var pushoverAppTokenAWSSecret string
	var awsProfile string
	var awsRegion string

	flag.BoolVar(&dryRun, "d", false, "Dry run only - so only messages are printed")
	flag.BoolVar(&sendToPushover, "sp", false, "Send notifications to pushover (set by default)")
	flag.BoolVar(&sendToSumo, "ss", false, "Send to Sumo")
	flag.StringVar(&userKey, "u", "", 
		fmt.Sprintf("Pushover User key, if not specified in env var: %s", PushoverUserKey))
	flag.StringVar(&appToken, "t", "", 
		fmt.Sprintf("Pushover App Token, if not specified in env var: %s", PushoverAppToken))
	flag.StringVar(&collectorURL, "scu", "", 
		fmt.Sprintf("Sumo collector URL, if not specified in env var: %s", SumoCollectorURL))
	flag.BoolVar(&pullSecretsFromAWS, "pa", false, "Pull the secrets from AWS")
	flag.StringVar(&pushoverUserKeyAWSSecret, "puka", PushoverUserKey, 
		"Name of the AWS Secrets Manager secret containing pushover user key")
	flag.StringVar(&pushoverAppTokenAWSSecret, "pata", PushoverAppToken,
		"Name of the AWS Secrets Manager secret containing pushover app Token")
	flag.StringVar(&collectorURLAWSSecret, "scua", SumoCollectorURL,
		"Name of the AWS Secrets Manager secret containing collector URL")
	flag.StringVar(&awsProfile, "ap", "",
		"Name of AWS Profile if pulling Pushover creds from AWS Secrets Manager")
	flag.StringVar(&awsRegion, "ar", "ap-southeast-2", 
		"AWS Region")
	flag.StringVar(&attachment, "a", "", "Attachment path")
	flag.IntVar(&timeout, "i", 8, "Chrome timeout to take screenshot for gowitness")
	flag.BoolVar(&parseSignature, "p", false,
		"Parse signature of format '[id]: url', and send screenshot if URL of form https://,http:// detected")
	flag.BoolVar(&verbose, "v", false, "Verbose message")
	flag.StringVar(&gowitness, "g", "gowitness",
		"Path to gowitness to take screenshot")
	flag.IntVar(&numThreads, "n", 3, "Number of threads")
	flag.StringVar(&screenshotRes, "r", "640,480", "Screenshot's resolution")
	flag.BoolVar(&sendUnique, "su", false, "Send unique requests only")
	flag.Parse()

	appToken = getAppToken(appToken, pushoverAppTokenAWSSecret, awsRegion, 
		awsProfile, pullSecretsFromAWS, sendToPushover)
	userKey = getUserKey(userKey, pushoverUserKeyAWSSecret, awsRegion, awsProfile, 
		pullSecretsFromAWS, sendToPushover)
	collectorURL = getSumoCollectorURL(collectorURL, collectorURLAWSSecret, awsRegion, 
		awsProfile, pullSecretsFromAWS, sendToSumo)

	// Checking if pushover user key and app token is available
	if sendToPushover && (appToken == "" || userKey == "") {
		log.Fatalf("[-] Both Pushover User key and App token must be provided")
	}	

	// Checking if Sumo collector URL and app token is available
	if sendToSumo && (collectorURL == "") {
		log.Fatalf("[-] Sumo collector URL must be provided")
	}	


	// Decide whether to display verbose log messages, or not
	if !verbose {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// Create a channel to hold the messages to process
	lines := make(chan string)

	// Lines previously sent should be captured
	linesSent := make(map[string]bool)

	// Create a new Pushover App with a token
	app := pushover.New(appToken)

	// Create a new recipient with the user key
	recipient := pushover.NewRecipient(userKey)

	// Build a random number generator
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	var wg sync.WaitGroup

	// Configure Resty client with content type/user agent string
	var restyClient *resty.Client
	if sendToSumo {
		restyClient = resty.New()
		configureResty(restyClient, UserAgentString)
	}

	// Start the goroutines to process the lines provided from channel
	for i := 0; i < numThreads; i++ {
		wg.Add(1)

		log.Printf("Started thread %d\n", i)

		go func() {
			defer wg.Done()

			for line := range lines {

				url := ""

				// Full Output file path to store screenshot
				outfile := ""

				// Build the output folder name
				outfolder := ScreenshotFolderPrefix + strconv.FormatInt(int64(r1.Intn(1000)), 10)

				// Build the output file name to store the screenshot
				outfileName := ScreenshotFilePrefix + strconv.FormatInt(int64(r1.Intn(1000)), 10) + ".png"

				if parseSignature {
					// Attempt to parse the signature into ID, and URL field
					m := getRegexGroups(`\[(?P<id>[a-zA-Z0-9\_\.\-]+)\]\s*(?P<url>.+)`,
						line)
					if len(m) > 0 {
						regexurl := m["url"]
						if isURLWithHTTPProtocol(regexurl) {
							url = regexurl

							log.Printf("Taking screenshot of URL: %s\n", url)
							outfile = takeScreenshot(url, outfolder, outfileName,
								gowitness, screenshotRes, timeout, dryRun)
						}
					}
				}

				// Track if lines have been previously sent if sendUnique flag
				// is set
				found := false
				if sendUnique {
					_, found = linesSent[line]
					linesSent[line] = true
				}

				// Send the message by pushover, if message not duplicated as
				// confirmed via anew
				if (sendUnique && !found) || !sendUnique {
					if sendToPushover {
						log.Printf("Sending message via Pushover: %s\n", line)
						sendMessageViaPushover(app, recipient, line, attachment, 
							outfile, dryRun)
					}

					if sendToSumo {
						sendMessageViaSumo(restyClient, collectorURL, line)
					}

					// Remove the screenshot as already sent to pushover
					if outfile != "" {
						log.Printf("[*] Removing screenshot outfile: %s\n", outfile)
						os.Remove(outfile)
					}

					// Remove the screenshot folder as well now that screenshot is sent
					if outfolder != "" {
						log.Printf("[*] Removing folder: %s\n", outfolder)
						os.Remove(outfolder)
					}

					// Print the input message as-is to output
					fmt.Println(line)
				}
			}
		}()
	}

	// Read signatures/lines to process from input
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			log.Printf("[*] Added line: %s for processing", line)
			lines <- line
		}
	}
	close(lines)

	// Wait for all goroutines to end
	wg.Wait()
}
