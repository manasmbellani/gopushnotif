// Function to send go push notification to pushover app.
// It also recognises special lines in the format [sig] urls if `-p` flag is set
// and takes a screenshot via gowitness (which should be installed in PATH dir)
//

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/gregdel/pushover"
)

const SCREENSHOTS_FOLDER = "out-screenshots-743837871"
const SCREENSHOT_FILE_NAME = "out-screenshot.png"

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
	} else {
		return false
	}
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

		cmd := exec.Command("/bin/bash", "-c", cmdToExec)
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
func takeScreenshot(url string, screenshotName string, gowitnessBin string,
	dryRun bool) string {

	// Output file path where screenshot is written
	outfile := ""

	// Default to screenshot.png if not specified
	if screenshotName == "" {
		screenshotName = "screenshot.png"
	}

	// Prepare screenshot template by clearing screenshots from the folder and
	// then take gowitness in the folder
	screenshotTmp := "rm {screenshots_folder}/*; "
	screenshotTmp += "{gowitness} single --url {url} -d {screenshots_folder}; "
	screenshotTmp += "mv {screenshots_folder}/*.png {screenshots_folder}/{screenshot_name}"

	// Create the screenshot folder if it doesn't exist
	_, err := os.Stat(SCREENSHOTS_FOLDER)
	if os.IsNotExist(err) {
		os.Mkdir(SCREENSHOTS_FOLDER, 0755)
	}

	if isURLWithHTTPProtocol(url) {
		// Take the screenshot
		screenshotCmd := screenshotTmp
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{screenshots_folder}",
			SCREENSHOTS_FOLDER)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{url}", url)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{screenshot_name}",
			SCREENSHOT_FILE_NAME)
		screenshotCmd = strings.ReplaceAll(screenshotCmd, "{gowitness}",
			gowitnessBin)
		outfile = path.Join(SCREENSHOTS_FOLDER, SCREENSHOT_FILE_NAME)

		execCmd(screenshotCmd, "", dryRun)

	}

	return outfile

}

// Worker function that will send a notification reading from the channel
//unc goSendNotf(app *pushover.Pushover, recipient *pushover.Pushover, msgs chan string, attachment string) {
//	for msg := range msgs {
//
//	}
//
//}

func main() {
	dryRunPtr := flag.Bool("d", false, "Dry run only - so only messages are printed")
	userKeyPtr := flag.String("u", "", "Pushover User key")
	appTokenPtr := flag.String("t", "", "Pushover App Token")
	attachmentPtr := flag.String("a", "", "Attachment path")
	parseSignaturePtr := flag.Bool("p", false,
		"Parse signature of format '[id]: url', and send screenshot if URL of form https://,http:// detected")
	verbosePtr := flag.Bool("v", false, "Verbose message")
	gowitnessPtr := flag.String("g", "gowitness",
		"Path to gowitness to take screenshot")
	flag.Parse()
	userKey := *userKeyPtr
	appToken := *appTokenPtr
	attachment := *attachmentPtr
	parseSignature := *parseSignaturePtr
	verbose := *verbosePtr
	gowitness := *gowitnessPtr
	dryRun := *dryRunPtr

	if appToken == "" {
		log.Fatalf("[-] App Token must be specified")
	}

	if userKey == "" {
		log.Fatalf("[-] User Key must be specified")
	}

	// Decide whether to display verbose log messages, or not
	if !verbose {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	// Create a new Pushover App with a token
	app := pushover.New(appToken)
	fmt.Println(reflect.TypeOf(app))

	// Create a new recipient with the user key
	recipient := pushover.NewRecipient(userKey)

	// Read signatures/lines to process from input
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {

			url := ""
			outfile := ""
			if parseSignature {
				// Attempt to parse the signature into ID, and URL field
				m := getRegexGroups(`\[(?P<id>[a-zA-Z0-9\_\.\-]+)\]\s*(?P<url>.+)`,
					line)
				if len(m) > 0 {
					regexurl := m["url"]
					if isURLWithHTTPProtocol(regexurl) {
						url = regexurl

						log.Printf("Taking screenshot of URL: %s\n", url)
						outfile = takeScreenshot(url, SCREENSHOT_FILE_NAME,
							gowitness, dryRun)
					}
				}
			}

			if !dryRun {
				// Create the message to send
				message := pushover.NewMessage(line)
				if attachment != "" {
					file, _ := os.Open(attachment)
					message.AddAttachment(bufio.NewReader(file))
				}

				// Attach the output file as well
				if outfile != "" {
					file, _ := os.Open(outfile)
					message.AddAttachment(bufio.NewReader(file))
				}

				// Send the image, and the response details too
				response, err := app.SendMessage(message, recipient)
				if err != nil {
					log.Panic(err)
				}

				// Print the Pushover API response
				log.Printf("Pushover API Response: %+v", response)
			}

			// Remove the screenshot as already sent to pushover
			if outfile != "" {
				log.Printf("Removing screenshot file: %s\n", outfile)
				os.Remove(outfile)
			}

			// Print the input message as-is to output
			fmt.Println(line)

		}
	}
}
