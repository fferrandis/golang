package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"./utils"
)

const (
	// BaseServer hyperdrive servers url
	BaseServer1 = "http://127.0.0.1:4244/"
	BaseServer2 = "http://127.0.0.1:4245/"
	BaseServer3 = "http://127.0.0.1:4246/"
	// BaseClient Hyperdrive client base url
	BaseClient         = "http://127.0.0.1:8889/"
	PortClient         = 8889
	maxFileDescriptors = 1000
)

// performPutGet
func performPutGet(client *http.Client, baseserver string, nrkeys int, payload string, maxChan chan bool, wg *sync.WaitGroup) {
	// Increment the number of goroutines to wait for
	wg.Add(1)

	// Store a random number to identify the current instance
	rand.Seed(time.Now().UTC().UnixNano())
	number := rand.Intn(1000)

	// defer wait group done
	defer log.Println("End of performPutGet ", number, baseserver)
	defer wg.Done()
	defer func(maxChan chan bool) { <-maxChan }(maxChan)

	for elt := 0; elt < nrkeys; elt++ {
		key := utils.GenerateKey(64)

		// Build PUT request
		putRequest := utils.PutKey(key, payload, baseserver)
		log.Println("Put key: ", key)
		res, err := client.Do(putRequest)

		if err != nil {
			log.Fatal(err)
		}

		if res.StatusCode != 204 {
			log.Fatal(res.StatusCode)
		}
		res.Body.Close()
		/*
			// Build GET request
			getRequest := utils.GetKey(key, baseserver)
			log.Println("Get key: ", key)
			res2, err2 := client.Do(getRequest)

			if res2.StatusCode != 200 {
				log.Fatal(err2)
			}
			res2.Body.Close()
		*/
	}
}

func getKeysIndex(client *http.Client, baseserver string) utils.ListKeys {
	var keys utils.ListKeys

	uri := baseserver + "info/index/key/list/"

	req, _ := http.NewRequest(http.MethodGet, uri, nil)

	req.Header.Set("Accept", "application/json")

	res, _ := client.Do(req)

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &keys)

	return keys
}

func getGroupsIndex(client *http.Client, baseserver string) utils.ListGroups {
	var groups utils.ListGroups

	uri := baseserver + "info/index/group/list/"

	req, _ := http.NewRequest(http.MethodGet, uri, nil)

	req.Header.Set("Accept", "application/json")

	res, _ := client.Do(req)
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &groups)

	return groups
}

// performPutGetClient hyperdrive client
func performPutGetClient(client *http.Client, baseclient string, nrkeys int, payload string, maxChan chan bool, wg *sync.WaitGroup) {
	// Increment the number of goroutines to wait for
	wg.Add(1)

	// Store a random number to identify the current instance
	rand.Seed(time.Now().UTC().UnixNano())
	number := rand.Intn(1000)

	// defer wait group done
	defer log.Println("End of performPutGet ", number, baseclient)
	defer wg.Done()
	defer func(maxChan chan bool) { <-maxChan }(maxChan)

	for elt := 0; elt < nrkeys; elt++ {
		key := fmt.Sprintf("dir-%d/obj-%d", elt, number)

		// Build PUT request
		putRequest := utils.PutKeyClient(key, payload, baseclient)
		log.Println("Put key: ", key)
		res, err := client.Do(putRequest)

		if err != nil {
			log.Fatal(err)
		}

		if res.StatusCode != 204 {
			log.Println(res)
			log.Println("Put key error: ", err)
		}
		res.Body.Close()
		/*
			// Build GET request
			getRequest := utils.GetKeyClient(key, baseclient)
			log.Println("Get key: ", key)
			res2, err2 := client.Do(getRequest)

			if res2.StatusCode != 200 {
				log.Println("Get key error:", err2)
			}
			res2.Body.Close()*/
	}
}

// performDelClient hyperdrive client
func performDelClient(client *http.Client, baseclient string, nrkeys int, wg *sync.WaitGroup) {
	wg.Add(1)

}

func mainServer(baseserver string, nrroutines int, payloadSize int, wgMain *sync.WaitGroup) {
	wgMain.Add(1)
	defer wgMain.Done()
	// Set values
	nrkeys := 1 // number of keys per routine
	payload := utils.RandomString(payloadSize)

	log.Println("Launch injector routines: ", nrroutines)

	// Create wait group object
	var wg sync.WaitGroup
	maxChan := make(chan bool, maxFileDescriptors)

	// HTTP client
	client := &http.Client{}

	start := time.Now().Unix()
	// Perform PUT & GET concurrently
	for i := 0; i < nrroutines; i++ {
		go performPutGet(client, baseserver, nrkeys, payload, maxChan, &wg)
	}

	wg.Wait()

	end := time.Now().Unix()

	log.Println(int(end) - int(start))

	keys := getKeysIndex(client, BaseServer1)

	log.Println(len(keys.Keys))
}

// mainClient perform http requests from hyperdrive client
func mainClient(baseclient string, nrroutines int, payloadSize int, wgMain *sync.WaitGroup) {
	wgMain.Add(1)
	defer wgMain.Done()

	client := &http.Client{}
	nrkeys := 1 // Number of keys per routine
	payload := utils.RandomString(payloadSize)
	// Create wait group object
	var wg sync.WaitGroup
	maxChan := make(chan bool, maxFileDescriptors)
	for i := 0; i < nrroutines; i++ {
		maxChan <- true
		go performPutGetClient(client, baseclient, nrkeys, payload, maxChan, &wg)
	}

	wg.Wait()

	grp1 := getGroupsIndex(client, BaseServer1)
	grp2 := getGroupsIndex(client, BaseServer2)
	grp3 := getGroupsIndex(client, BaseServer3)

	log.Println("Groups hd1: ", len(grp1.Groups))
	log.Println("Groups hd2: ", len(grp2.Groups))
	log.Println("Groups hd3: ", len(grp3.Groups))
}

func main() {

	// Arguments
	workersPtr := flag.Int("workers", 64, "number of workers in parallel")
	typePtr := flag.String("hd-type", "server", "Choose between hyperdrive 'server' or 'client'")
	payloadSizePtr := flag.Int("size", 1024*1024, "Payload size")
	nrclientPtr := flag.Int("nrclients", 1, "number of HD clients")

	flag.Parse()

	var wgMain sync.WaitGroup

	// Main call
	if *typePtr == "server" {
		go mainServer(BaseServer1, *workersPtr, *payloadSizePtr, &wgMain)
	} else if *typePtr == "client" {
		for nrclient := 0; nrclient < *nrclientPtr; nrclient++ {
			port := PortClient + nrclient
			baseclient := "http://127.0.0.1:" + strconv.Itoa(port) + "/"
			go mainClient(baseclient, *workersPtr, *payloadSizePtr, &wgMain)
		}

	}
	wgMain.Wait()

}
