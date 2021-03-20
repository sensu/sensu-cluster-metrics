package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	"github.com/sensu/sensu-go/types"
	"github.com/shurcooL/graphql"
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
			Default:   "http://localhost:3000/",
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

//{ versions { backend { version } } }
var query struct {
	Versions struct {
		Backend struct {
			Version graphql.String
		}
	}
}

/* This should match graphiql query:
query ClusterMetrics {
    forward {
      __typename
      ... on Query {
      	clusterName
      }

    }
}

*/

// golang client Ref: https://github.com/shurcooL/graphql#readme
var federationQuery struct { // Top level golang var representing the query
	ClusterMetrics struct {
		forward struct {
			__typename graphql.String
			Query      struct {
				clusterName graphql.String
			} `graphql:"... on Query"` //instruct golang client that this is an inline fragment
		}
	}
}

/* TODO: the rest of the query structure to add
clusterMetrics struct {
	namespaces struct {
		name        graphql.String
		eventGauges struct {
			total          graphql.Int
			statusCritical graphql.Int
			statusWarning  graphql.Int
			statusOther    graphql.Int
			statusOK       graphql.Int
			statePassing   graphql.Int
			stateFailing   graphql.Int
		}
		keepaliveGauges struct {
			total    graphql.Int
			statusOK graphql.Int
		}
		entityGauges struct {
			total graphql.Int
			agent graphql.Int
			proxy graphql.Int
			other graphql.Int
		}
	}
	clusterGauges struct {
		total graphql.Int
	}
} `graphql:"... on ClusterMetrics"`
*/

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

func executeCheck(event *types.Event) (int, error) {
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
		Transport: &transport{
			underlyingTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: plugin.SkipVerify,
					RootCAs:            roots,
				},
			},
		},
	}

	gqlclient := graphql.NewClient("https://localhost:3000/graphql", httpclient)
	err := gqlclient.Query(context.Background(), &query, nil)
	if err != nil {
		fmt.Printf("Error doing query: %v", err)
	}
	fmt.Println(query.Versions.Backend.Version)
	err = gqlclient.Query(context.Background(), &federationQuery, nil)
	if err != nil {
		fmt.Printf("Error doing query: %v", err)
	}
	return sensu.CheckStateOK, nil
}

type transport struct {
	underlyingTransport http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(plugin.ApiKey) > 0 {
		req.Header.Add("Authorization", "Key "+plugin.ApiKey)
	}
	return t.underlyingTransport.RoundTrip(req)
}
