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
	"math/rand"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gregdel/pushover"
)

// SCREENSHOTS_FOLDER - Folder to store the screenshot
const ScreenshotFolderPrefix = "out-screenshots"

// SCREENSHOT_FILE_PREFIX - Prefix for the screenshot
const ScreenshotFilePrefix = "out-screenshot"

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
	timeoutPtr := flag.Int("i", 8, "Chrome timeout to take screenshot for gowitness")
	parseSignaturePtr := flag.Bool("p", false,
		"Parse signature of format '[id]: url', and send screenshot if URL of form https://,http:// detected")
	verbosePtr := flag.Bool("v", false, "Verbose message")
	gowitnessPtr := flag.String("g", "gowitness",
		"Path to gowitness to take screenshot")
	numThreadsPtr := flag.Int("n", 3, "Number of threads")
	screenshotResPtr := flag.String("r", "640,480", "Screenshot's resolution")
	flag.Parse()
	userKey := *userKeyPtr
	appToken := *appTokenPtr
	attachment := *attachmentPtr
	parseSignature := *parseSignaturePtr
	verbose := *verbosePtr
	gowitness := *gowitnessPtr
	dryRun := *dryRunPtr
	numThreads := *numThreadsPtr
	screenshotRes := *screenshotResPtr
	timeout := *timeoutPtr

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

	// Create a channel to hold the messages to process
	lines := make(chan string)

	// Create a new Pushover App with a token
	app := pushover.New(appToken)

	// Create a new recipient with the user key
	recipient := pushover.NewRecipient(userKey)

	// Build a random number generator
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	var wg sync.WaitGroup

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

				log.Printf("Sending message: %s\n", line)
				if !dryRun {
					// Create the message to send
					message := pushover.NewMessage(line)
					if attachment != "" {
						file, _ := os.Open(attachment)
						message.AddAttachment(bufio.NewReader(file))
					}

					// Attach the output file as well
					if outfile != "" {

						// First check if it even exists
						_, err := os.Stat(outfile)
						if !os.IsNotExist(err) {
							file, _ := os.Open(outfile)
							message.AddAttachment(bufio.NewReader(file))
						} else {
							log.Printf("[!] File: %s did not exist. Can't send screenshot.", outfile)
						}
					}

					// Send the image, and the response details too
					response, err := app.SendMessage(message, recipient)
					if err != nil {
						log.Println(err)
					}

					// Print the Pushover API response
					log.Printf("Pushover API Response: %+v", response)
				}

				// Remove the screenshot as already sent to pushover
				if outfile != "" {
					log.Printf("Removing screenshot outfile: %s\n", outfile)
					os.Remove(outfile)
				}

				// Remove the screenshot folder as well now that screenshot is sent
				if outfolder != "" {
					log.Printf("Removing folder: %s\n", outfolder)
					os.Remove(outfolder)
				}

				// Print the input message as-is to output
				fmt.Println(line)
			}
		}()
	}

	// Read signatures/lines to process from input
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		line := sc.Text()
		if line != "" {
			log.Printf("Added line: %s for processing", line)
			lines <- line
		}
	}
	close(lines)

	// Wait for all goroutines to end
	wg.Wait()
}
