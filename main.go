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
	"strconv"
	"strings"
	"time"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	Url          string
	ApiKey       string
	CAFile       string
	OutputFormat string
	SkipVerify   bool
}

var metrics = []string{}
var tags = map[string]string{}

var (
	clusterQueryData = map[string]string{"query": `{
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
			Name:     "sensu-cluster-metrics",
			Short:    "Sensu cluster metrics using graphQL",
			Keyspace: "sensu.io/plugins/sensu-cluster-metrics/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "url",
			Env:       "CLUSTER_URL",
			Argument:  "url",
			Shorthand: "u",
			Default:   "http://localhost:8080/graphql",
			Usage:     "url to access the Sensu cluster graphql endpoint",
			Value:     &plugin.Url,
		},
		&sensu.PluginConfigOption{
			Path:      "api-key",
			Env:       "CLUSTER_API_KEY",
			Argument:  "api-key",
			Shorthand: "k",
			Secret:    true,
			Default:   "",
			Usage:     "Sensu apikey for authentication (use envvar CLUSTER_API_KEY in production)",
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
			Path:     "output-format",
			Argument: "output-format",
			Env:      "CLUSTER_OUTPUT_FORMAT",
			Default:  "opentsdb_line",
			Usage:    "metrics output format, supports: opentsdb_line or prometheus_text",
			Value:    &plugin.OutputFormat,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.Url) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--url or CLUSTER_URL environment variable is required")
	}
	if len(plugin.ApiKey) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--apikey or CLUSTER_API_KEY environment variable is required")
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

type clusterMetricsQuery struct {
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
	timeNow := time.Now().Unix()
	jsonData := map[string]string{
		"query": `{ versions { backend { version } } }`,
	}
	data, err := graphqlQuery(jsonData)
	if err != nil {
		fmt.Printf("The version query HTTP request failed with error %s\n", err)
		return sensu.CheckStateCritical, nil
	}
	var vresult versionQuery
	err = json.Unmarshal([]byte(data), &vresult)
	if err != nil {
		fmt.Printf("Version query response data parsing failed with error %s\n", err)
		return sensu.CheckStateCritical, nil
	}
	if len(vresult.Data.Versions.Backend.Version) > 0 {
		tags["sensu_backend_version"] = vresult.Data.Versions.Backend.Version
		tags["sensu_backend_url"] = plugin.Url
	} else {
		fmt.Printf("Unable to find Sensu backend version something went wrong\n")
		return sensu.CheckStateCritical, nil
	}
	data, err = graphqlQuery(clusterQueryData)
	if err != nil {
		fmt.Printf("The cluster query HTTP request failed with error %s\n", err)
		return sensu.CheckStateCritical, nil
	}
	var qresult clusterMetricsQuery
	err = json.Unmarshal([]byte(data), &qresult)
	if err != nil {
		fmt.Printf("Cluster metrics query response data parsing failed with error %s\n", err)
		return sensu.CheckStateCritical, nil
	}
	for _, cluster := range qresult.Data.Clusters {
		if strings.Compare(cluster.Name, "~") == 0 {
			tags["cluster"] = "local"
		} else {
			tags["cluster"] = cluster.Name
		}
		addMetric("cluster.total", tags, strconv.Itoa(cluster.Metrics.ClusterGauges.Total), timeNow)
		for _, ns := range cluster.Metrics.Namespaces {
			if strings.Compare(ns.Name, "~") == 0 {
				tags["namespace"] = "local"
			} else {
				tags["namespace"] = ns.Name
			}
			addMetric("namespace.entity.total", tags, strconv.Itoa(ns.EntityGauges.Total), timeNow)
			addMetric("namespace.entity.agent", tags, strconv.Itoa(ns.EntityGauges.Agent), timeNow)
			addMetric("namespace.entity.other", tags, strconv.Itoa(ns.EntityGauges.Other), timeNow)
			addMetric("namespace.entity.proxy", tags, strconv.Itoa(ns.EntityGauges.Proxy), timeNow)

			addMetric("namespace.keepalive.total", tags, strconv.Itoa(ns.KeepaliveGauges.Total), timeNow)
			addMetric("namespace.keepalive.state.passing", tags, strconv.Itoa(ns.KeepaliveGauges.StatePassing), timeNow)
			addMetric("namespace.keepalive.state.failing", tags, strconv.Itoa(ns.KeepaliveGauges.StateFailing), timeNow)
			addMetric("namespace.keepalive.status.okay", tags, strconv.Itoa(ns.KeepaliveGauges.StatusOK), timeNow)
			addMetric("namespace.keepalive.status.warning", tags, strconv.Itoa(ns.KeepaliveGauges.StatusWarning), timeNow)
			addMetric("namespace.keepalive.status.critical", tags, strconv.Itoa(ns.KeepaliveGauges.StatusCritical), timeNow)
			addMetric("namespace.keepalive.status.other", tags, strconv.Itoa(ns.KeepaliveGauges.StatusOther), timeNow)

			addMetric("namespace.event.total", tags, strconv.Itoa(ns.EventGauges.Total), timeNow)
			addMetric("namespace.event.state.passing", tags, strconv.Itoa(ns.EventGauges.StatePassing), timeNow)
			addMetric("namespace.event.state.failing", tags, strconv.Itoa(ns.EventGauges.StateFailing), timeNow)
			addMetric("namespace.event.status.okay", tags, strconv.Itoa(ns.EventGauges.StatusOK), timeNow)
			addMetric("namespace.event.status.warning", tags, strconv.Itoa(ns.EventGauges.StatusWarning), timeNow)
			addMetric("namespace.event.status.critical", tags, strconv.Itoa(ns.EventGauges.StatusCritical), timeNow)
			addMetric("namespace.event.status.other", tags, strconv.Itoa(ns.EventGauges.StatusOther), timeNow)
		}
	}
	for _, metric := range metrics {
		fmt.Println(metric)
	}

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

func addMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	switch plugin.OutputFormat {
	case "opentsdb_line":
		addOpenTSDBMetric(metricName, tags, value, timeNow)
	case "prometheus_text":
		addPrometheusMetric(metricName, tags, value, timeNow)
	default:
		addOpenTSDBMetric(metricName, tags, value, timeNow)
	}
}

func addOpenTSDBMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	tagStr := ""
	for tag, tvalue := range tags {
		if len(tagStr) > 0 {
			tagStr = tagStr + " "
		}
		tagStr = tagStr + tag + "=" + tvalue
	}
	outputs := []string{metricName, strconv.FormatInt(timeNow, 10), value, tagStr}
	//fmt.Println(strings.Join(outputs, " "))
	metrics = append(metrics, strings.Join(outputs, " "))
}

func addPrometheusMetric(metricName string, tags map[string]string, value string, timeNow int64) {
	tagStr := ""
	for tag, tvalue := range tags {
		if len(tagStr) > 0 {
			tagStr = tagStr + ","
		}
		tagStr = tagStr + tag + "=\"" + tvalue + "\""
	}
	if len(tagStr) > 0 {
		tagStr = "{" + tagStr + "}"
	}
	metricName = strings.Replace(metricName, ".", "_", -1)
	outputs := []string{metricName + tagStr, value, strconv.FormatInt(timeNow, 10)}
	//fmt.Println(strings.Join(outputs, " "))
	metrics = append(metrics, strings.Join(outputs, " "))
}
