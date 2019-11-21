package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultRetryInterval = time.Second
const defaultRetryBackoffMaxInterval = 300 * time.Second
var retryBackoffMaxInterval = defaultRetryBackoffMaxInterval

type sliceVar []string
type hostFlagsVar []string

type Context struct {
}

type HttpHeader struct {
	name  string
	value string
}

func (c *Context) Env() map[string]string {
	env := make(map[string]string)
	for _, i := range os.Environ() {
		sep := strings.Index(i, "=")
		env[i[0:sep]] = i[sep+1:]
	}
	return env
}

var (
	buildVersion 		string
	version      		bool
	poll	 		bool
	wg	   		sync.WaitGroup

	headersFlag     	sliceVar
	presentFlag     	sliceVar
	absentFlag		sliceVar
	headers	   		[]HttpHeader
	urls	      		[]url.URL
	present	   		[]string
	absent	    		[]string
	hostFlag		hostFlagsVar
	retryInterval 		time.Duration
	retryBackoffFlag 	bool
	insecureFlag 		bool
	timeoutFlag   		time.Duration
	dependencyChan  	chan struct{}

)

func (i *hostFlagsVar) String() string {
	return fmt.Sprint(*i)
}

func (i *hostFlagsVar) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (s *sliceVar) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func (s *sliceVar) String() string {
	return strings.Join(*s, ",")
}

func awaitForDependencies() {
	dependencyChan := make(chan struct{})

	go func() {
		for _, u := range urls {
			log.Println("Awaiting for:", u.String())

			switch u.Scheme {
			case "file":
				wg.Add(1)
				go func(u url.URL) {
					defer wg.Done()
					ticker := time.NewTicker(retryInterval)
					defer ticker.Stop()
					var err error
					for range ticker.C {
						if _, err = os.Stat(u.Path); err == nil {
							log.Printf("File %s had been generated\n", u.String())
							return
						} else if os.IsNotExist(err) {
							continue
						} else {
							log.Printf("Problem with check file %s exist: %v. Sleeping %s\n", u.String(), err.Error(), retryInterval)
						}
					}
				}(u)
			case "tcp", "tcp4", "tcp6":
				awaitForSocket(u.Scheme, u.Host, timeoutFlag)
			case "unix":
				awaitForSocket(u.Scheme, u.Path, timeoutFlag)
			case "http", "https":
				wg.Add(1)
				go func(u url.URL) {
					tr := &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureFlag},
					}
					client := &http.Client{
						Timeout: timeoutFlag,
						Transport: tr,
					}
					
					defer wg.Done()
					for {
						req, err := http.NewRequest("GET", u.String(), nil)
						if err != nil {
							log.Printf("Problem with dial: %v. Sleeping %s\n", err.Error(), retryInterval)
							time.Sleep(retryInterval)
						}
						if len(headers) > 0 {
							for _, header := range headers {
								req.Header.Add(header.name, header.value)
							}
						}

						resp, err := client.Do(req)
						if err != nil {
							log.Printf("Problem with request: %s. Sleeping %s\n", err.Error(), retryInterval)
							time.Sleep(retryInterval)
						} else if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
							log.Printf("Received %d from %s\n", resp.StatusCode, u.String())
							if len(present) == 0 && len(absent) == 0 {
								return
							}
							defer resp.Body.Close()
							body, err := ioutil.ReadAll(resp.Body)
							if err != nil {
								log.Printf("Problem reading request body: %s", err.Error())
							}
							// log.Printf("BODY: %s\n", string(body))
							var requiredTextNotFound = 0
							var forbiddenTextFound = 0
							if len(present) > 0 {
								for _, p := range present {
									//log.Printf("PRESENT TEXT: %s\n", p)
									if strings.Contains(string(body), p) {
										log.Printf(" - Found required text: %s\n", p)
									} else {
										log.Printf(" - Required text not found: %s\n", p)
										requiredTextNotFound = 1
									}
								}
							}
							if len(absent) > 0 {
								for _, a := range absent {
									//log.Printf("ABSENT TEXT: %s\n", a)
									if strings.Contains(string(body), a) {
										log.Printf(" - Found forbidden text: %s\n", a)
										forbiddenTextFound = 1
									} else {
										log.Printf(" - Forbidden text not found: %s\n", a)
									}
								}
							}
							if requiredTextNotFound == 0 && forbiddenTextFound == 0 {
								return
							} else {
								if requiredTextNotFound == 1 {
									log.Printf("NOT all required text was found.")
								}
								if forbiddenTextFound == 1 {
									log.Printf("At least some forbidden text was found.")
								}
								log.Printf("Sleeping %s\n", retryInterval)
								time.Sleep(retryInterval)
							}
						} else {
							log.Printf("Received %d from %s. Sleeping %s\n", resp.StatusCode, u.String(), retryInterval)
							time.Sleep(retryInterval)
						}
						if retryBackoffFlag == true {
							retryInterval += retryInterval
							if retryInterval > retryBackoffMaxInterval {
								retryInterval = retryBackoffMaxInterval
							}
						}
					}
				}(u)
			default:
				log.Fatalf("invalid host protocol provided: %s. supported protocols are: tcp, tcp4, tcp6 and http", u.Scheme)
			}
		}
		wg.Wait()
		close(dependencyChan)
	}()

	select {
	case <-dependencyChan:
		break
	case <-time.After(timeoutFlag):
		log.Fatalf("Timeout after %s awaiting on dependencies to become available: %v", timeoutFlag, hostFlag)
	}

}

func awaitForSocket(scheme, addr string, timeout time.Duration) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := net.DialTimeout(scheme, addr, timeoutFlag)
			if err != nil {
				log.Printf("Problem with dial: %v. Sleeping %s\n", err.Error(), retryInterval)
				time.Sleep(retryInterval)
			}
			if retryBackoffFlag == true {
				retryInterval += retryInterval
				if retryInterval > retryBackoffMaxInterval {
					retryInterval = retryBackoffMaxInterval
				}
			}
			if conn != nil {
				log.Printf("Connected to %s://%s\n", scheme, addr)
				return
			}
		}
	}()
}

func usage() {
	println(`Usage: await [options] [command]

Utility to wait for a socket, an http(s) response or a file before launching a command

Options:`)
	flag.PrintDefaults()

	println(`
Arguments:
  command - command to be executed
  `)

	println(`Examples:
`)
	println(` - Wait for a database to become available on port 5432 and start nginx.`)
	println(`     await -url tcp://db:5432 nginx
`)
	println(` - Wait for a website to become available on port 8000 and start nginx.`)
	println(`     await -url http://web:8000 nginx
`)
	println(` - Wait 90s for a website to become available on port 38383, look for text "ready" and make sure text "fail" is not present. Retry after 5,10,20,40,80,80,etc. seconds and start nginx.`)
	println(`     await --url http://localhost:38383 --text-present "ready" --text-absent "fail" --timeout 300s --retry-interval 5s --retry-backoff --retry-backoff-max-interval 80s
`)

	println(`For more information, see https://github.com/treksler/await`)
}

func main() {

	flag.BoolVar(&version, "version", false, "show version")

	flag.Var(&headersFlag, "http-header", "HTTP headers, colon separated. e.g \"Accept-Encoding: gzip\". Can be passed multiple times")
	flag.Var(&presentFlag, "text-present", "Text required text to be present in HTTP response body. Can be passed multiple times")
	flag.Var(&absentFlag, "text-absent", "Text required to be absent from HTTP response body. Can be passed multiple times")
	flag.Var(&hostFlag, "url", "Host (tcp/tcp4/tcp6/http/https/unix/file) to await before this container starts. Can be passed multiple times. e.g. tcp://db:5432")
	flag.DurationVar(&timeoutFlag, "timeout", 10*time.Second, "URL wait timeout")
	flag.DurationVar(&retryInterval, "retry-interval", defaultRetryInterval, "Duration to wait before retrying")
      	flag.BoolVar(&retryBackoffFlag, "retry-backoff", false, "Double the retry time, with each iteration. (default: false)")
	flag.BoolVar(&insecureFlag, "http-insecure", false, "Allow connections to HTTPS sites without valid certs. (default: false)")
	flag.DurationVar(&retryBackoffMaxInterval, "retry-backoff-max-interval", defaultRetryBackoffMaxInterval, "Maximum duration to wait before retrying, when retry backoff is enabled.")


	flag.Usage = usage
	flag.Parse()

	if version {
		fmt.Println(buildVersion)
		return
	}

	if retryBackoffFlag == true && retryInterval > retryBackoffMaxInterval {
		log.Printf("Retry Interval %s exceeds maximum backoff retry interval of %s. Using %s\n", retryInterval, retryBackoffMaxInterval, retryBackoffMaxInterval)
		retryInterval = retryBackoffMaxInterval
	}

	if flag.NArg() == 0 && flag.NFlag() == 0 {
		usage()
		os.Exit(1)
	}

	for _, p := range presentFlag {
		present = append(present, p)
	}

	for _, a := range absentFlag {
		absent = append(absent, a)
	}

	for _, host := range hostFlag {
		u, err := url.Parse(host)
		if err != nil {
			log.Fatalf("bad hostname provided: %s. %s", host, err.Error())
		}
		urls = append(urls, *u)
	}

	for _, h := range headersFlag {
		//validate headers need -host options
		if len(hostFlag) == 0 {
			log.Fatalf("-http-header \"%s\" provided with no -host option", h)
		}

		const errMsg = "bad HTTP Headers argument: %s. expected \"headerName: headerValue\""
		if strings.Contains(h, ":") {
			parts := strings.Split(h, ":")
			if len(parts) != 2 {
				log.Fatalf(errMsg, headersFlag)
			}
			headers = append(headers, HttpHeader{name: strings.TrimSpace(parts[0]), value: strings.TrimSpace(parts[1])})
		} else {
			log.Fatalf(errMsg, headersFlag)
		}

	}

	awaitForDependencies()

	if flag.NArg() > 0 {
		wg.Add(1)
		go Exec(flag.Arg(0), flag.Args())
	}

	wg.Wait()
}
