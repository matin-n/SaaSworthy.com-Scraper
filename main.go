package main

import (
	"bufio"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"mvdan.cc/xurls"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strconv"
	"strings"
)

func main() {

	cookieJar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar: cookieJar,
	}

	categories, err := readLines("categories.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	for _, category := range categories { // iterate thru every category
		totalCount := bruteforcePageCount(category, *client) // returns total pages of category
		for i := 1; i < totalCount+1; i++ {
			listURL := grabListings(category, i, *client) // returns slice containing URLs to scrape from
			for _, url := range listURL {
				fmt.Println(gatherInformation(url, *client)) // grab information from listing
			}
		}
	}

}

func bruteforcePageCount(category string, client http.Client) int {
	pageNumber := 0
	baseURL := "https://www.saasworthy.com/list/"

	for {
		//https://www.saasworthy.com/list/email-marketing-software
		resp, err := client.Get(baseURL + category + "?page=" + strconv.Itoa(pageNumber))
		if err != nil {
			fmt.Println(err)
		}

		resp, err = client.Get(baseURL + category + "?page=" + strconv.Itoa(pageNumber))
		if err != nil {
			fmt.Println(err)
		}

		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}

		bodyString := string(bodyBytes)

		// this can cause an infinite loop on errors...works as a quick hack
		if strings.Contains(bodyString, "Sorry, No Result Found.") {
			return pageNumber - 1
		} else {
			pageNumber = pageNumber + 1
			//fmt.Println(strconv.Itoa(pageNumber))
		}
	}
}

func grabListings(category string, pageNum int, client http.Client) []string {

	baseURL := "https://www.saasworthy.com/list/"

	resp, err := client.Get(baseURL + category + "?page=" + strconv.Itoa(pageNum))
	if err != nil {
		fmt.Println(err)
	}

	resp, err = client.Get(baseURL + category + "?page=" + strconv.Itoa(pageNum))
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	bodyString := string(bodyBytes)

	var fullList []string
	var urlList []string

	rxStrict := xurls.Strict()
	fullList = rxStrict.FindAllString(bodyString, -1)
	fullList = removeDuplicatesUnordered(fullList)

	for _, value := range fullList {
		if strings.Contains(value, "saasworthy.com/product/") {
			urlList = append(urlList, value)
		}
	}

	return urlList
}

func gatherInformation(url string, client http.Client) (string, string, string, string) {

	resp, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	bodyString := string(bodyBytes)

	quote := `"`

	// get company category and company name
	trimLeft := "}	</script>\n        \n        <script type=" + quote + "application/ld+json" + quote + ">\n       "
	trimRight := "]}	</script>"
	data_category_name := GetStringInBetween(bodyString, trimLeft, trimRight) + "]}"
	companyCategory := gjson.Get(data_category_name, "itemListElement.1.item.name").String()
	companyName := gjson.Get(data_category_name, "itemListElement.2.item.name").String()

	// get company URL
	trimLeft = "from the <a target=" + quote + "_blank" + quote + " href=" + quote
	trimRight = quote + " rel=\\" + quote + "nofollow\\" + quote + quote + ">vendor website"
	companyURL := GetStringInBetween(bodyString, trimLeft, trimRight) //TODO: follow redirect on domain to get actual domain
	companyURL = followURL(companyURL, client)

	// get company LinkedIn
	trimLeft = ">FOLLOWERS</div>\n                                <a target=" + quote + "_blank" + quote + " rel=" + quote + "nofollow" + quote + " href=" + quote
	trimRight = quote + "><div class=" + quote + "flwrs-row" + quote + ">"
	companyLinkedIn := GetStringInBetween(bodyString, trimLeft, trimRight)

	/*	fmt.Println("Company Name:", companyName)
		fmt.Println("Company Category:", companyCategory)
		fmt.Println("Company URL:", companyURL)
		fmt.Println("LinkedIn:", companyLinkedIn+"\n")
	*/
	return companyName, companyCategory, companyURL, companyLinkedIn
}

func followURL(url string, client http.Client) string {
	if url != "" {
		resp, err := client.Get(url)
		if err != nil {
			fmt.Println(err)
		}

		defer resp.Body.Close()

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
		}

		bodyString := string(bodyBytes)

		quote := `"`
		trimLeft := "div id=" + quote + "sendingMsg" + quote + ">On your way to <strong>"
		trimRight := "</strong></div>"
		companyURL := GetStringInBetween(bodyString, trimLeft, trimRight) //TODO: follow redirect on domain to get actual domain
		return companyURL
	} else {
		return ""
	}
}

func GetStringInBetween(str string, start string, end string) (result string) {
	// SOURCE: https://stackoverflow.com/a/42331558/9393975
	s := strings.Index(str, start)
	if s == -1 {
		return
	}
	s += len(start)
	e := strings.Index(str, end)
	if e == -1 {
		return
	}
	return str[s:e]
}

// read line by line into memory
// all file contents is stores in lines[]
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func removeDuplicatesUnordered(elements []string) []string {
	encountered := map[string]bool{}

	// Create a map of all unique elements.
	for v := range elements {
		encountered[elements[v]] = true
	}

	// Place all keys from the map into a slice.
	result := []string{}
	for key, _ := range encountered {
		result = append(result, key)
	}
	return result
}
