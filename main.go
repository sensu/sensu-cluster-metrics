package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	"io/ioutil"
	"log"
	"net/http"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Url        string
	ApiKey     string
	CAFile     string
	SkipVerify bool
}

var (
	fedQueryData = map[string]string{"query": `{
  clusters: forward {
    ... on Query {
      name: clusterName
      metrics: clusterMetrics {
        ... on ClusterMetrics {
          namespaces {
            name
            eventGauges {
              total
              statusCritical
              statusWarning
              statusOther
              statusOK
              statePassing
              stateFailing
            }
            keepaliveGauges {
              total
              statusCritical
              statusWarning
              statusOther
              statusOK
              statePassing
              stateFailing
            }
            entityGauges {
              total
              agent
              proxy
              other
            }
          }
          clusterGauges {
            total
          }
        }
        ... on FetchErr {
          code
          message
        }
      }
    }
    ... on ForwardErr {
      errName: name
      errMsg: err
    }
  }}`}
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-dashboard-metrics",
			Short:    "Sensu dashboard metrics using graphQL",
			Keyspace: "sensu.io/plugins/sensu-dashboard-metrics/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "url",
			Env:       "DASHBOARD_URL",
			Argument:  "url",
			Shorthand: "u",
			Default:   "http://localhost:3000/graphql",
			Usage:     "Url to access the Sensu Dashboard Web-UI",
			Value:     &plugin.Url,
		},
		&sensu.PluginConfigOption{
			Path:      "apikey",
			Env:       "DASHBOARD_APIKEY",
			Argument:  "apikey",
			Shorthand: "a",
			Secret:    true,
			Default:   "",
			Usage:     "Sensu apikey for authentication (use envvar DASHBOARD_APIKEY in production)",
			Value:     &plugin.ApiKey,
		},
		&sensu.PluginConfigOption{
			Path:     "skip-insecure-verify",
			Argument: "skip-insecure-verify",
			Default:  false,
			Usage:    "skip TLS certificate verification (not recommended!)",
			Value:    &plugin.SkipVerify,
		},
		&sensu.PluginConfigOption{
			Path:     "ca-file",
			Argument: "ca-file",
			Env:      "DASHBOARD_CA_FILE",
			Default:  "",
			Usage:    "TLS CA certificate bundle in PEM format",
			Value:    &plugin.CAFile,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.Url) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--url or DASHBOARD_URL environment variable is required")
	}
	if len(plugin.ApiKey) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--apikey or DASHBOARD_APIKEY environment variable is required")
	}
	return sensu.CheckStateOK, nil
}

type versionQuery struct {
	Data struct {
		Versions struct {
			Backend struct {
				Version string
			}
		}
	}
}

type entityGauges struct {
	Agent int
	Other int
	Proxy int
	Total int
}

type eventGauges struct {
	StateFailing   int
	StatePassing   int
	StatusCritical int
	StatusOK       int
	StatusOther    int
	StatusWarning  int
	Total          int
}

type keepaliveGauges struct {
	StateFailing   int
	StatePassing   int
	StatusCritical int
	StatusOK       int
	StatusOther    int
	StatusWarning  int
	Total          int
}

type federationMetricsQuery struct {
	Auth bool
	Data struct {
		Clusters []struct {
			Name    string
			Metrics struct {
				ClusterGauges struct {
					Total int
				}
				Namespaces []struct {
					Name            string
					EntityGauges    entityGauges
					EventGauges     eventGauges
					KeepaliveGauges keepaliveGauges
				}
			}
			ErrName string
			ErrMsg  string
		}
	}
}

func executeCheck(event *types.Event) (int, error) {
	//version query
	jsonData := map[string]string{
		"query": `{ versions { backend { version } } }`,
	}
	data, err := graphqlQuery(jsonData)
	if err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return sensu.CheckStateCritical, nil
	}
	var result versionQuery
	json.Unmarshal([]byte(data), &result)
	fmt.Printf("result: %+v\n", result)

	data, err = graphqlQuery(fedQueryData)
	var fresult federationMetricsQuery
	json.Unmarshal([]byte(data), &fresult)
	fmt.Printf("fresult: %+v\n", fresult)
	return sensu.CheckStateOK, nil
}

func graphqlQuery(queryStr map[string]string) ([]byte, error) {
	roots := x509.NewCertPool()
	if len(plugin.CAFile) > 0 {
		rootPEM, err := ioutil.ReadFile(plugin.CAFile)
		if err != nil {
			log.Fatal(err)
		}
		ok := roots.AppendCertsFromPEM([]byte(rootPEM))
		if !ok {
			log.Fatalf("Failed to parse root certificate at %v", plugin.CAFile)
		}
	}
	httpclient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: plugin.SkipVerify,
				RootCAs:            roots,
			},
		},
	}
	jsonValue, _ := json.Marshal(queryStr)
	//fmt.Println(string(jsonValue))
	request, err := http.NewRequest("POST", plugin.Url, bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("The HTTP request build failed with error %s\n", err)
		return nil, err
	}
	request.Header.Add("Authorization", "Key "+plugin.ApiKey)
	request.Header.Add("Content-Type", "application/json")
	response, err := httpclient.Do(request)
	if response == nil || err != nil {
		fmt.Printf("The HTTP request failed with error %s\n", err)
		return nil, err
	}
	defer response.Body.Close()
	//fmt.Println(response.Status)

	data, _ := ioutil.ReadAll(response.Body)
	//fmt.Println(string(data))
	return []byte(data), err
}
