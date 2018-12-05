package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rwn3120/ci-pipelines/gitlab"
)

const (
	envGitlabProjectsCSV    = "GITLAB_PROJECTS_CSV"
	envGitlabProtocol       = "GITLAB_PROTOCOL"
	gitlabProtocolDefault   = "https"
	gitlabProtocolProperty  = "protocol"
	envGitlabHost           = "GITLAB_HOST"
	gitlabHostDefault       = "127.0.0.1"
	gitlabHostProperty      = "host"
	envGitlabAPI            = "GITLAB_API_VERSION"
	gitlabAPIDefault        = 3
	gitlabAPIProperty       = "version"
	envGitlabToken          = "GITLAB_TOKEN"
	gitlabTokenProperty     = "token"
	envRefreshInterval      = "REFRESH_INTERVAL"
	refreshIntervalDefault  = 30
	refreshIntervalProperty = "refresh"
	envListenAddr           = "LISTEN_ADDR"
	listenAddrDefault       = "0.0.0.0:1111"
	listenAddrProperty      = "listen"
	envCount                = "COUNT"
	countDefault            = -1
	countProperty           = "count"
	envHistory              = "HISTORY"
	historyDefault          = 5
	historyProperty         = "history"
)

var bin = filepath.Base(os.Args[0])
var stderr = os.Stderr

func isSet(value string) bool {
	return len(strings.TrimSpace(value)) > 0
}

func printErrf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func printErrln(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func check(err error) {
	if err != nil {
		printErrln("Error:", err)
		os.Exit(254)
	}
}

func clear() {
	fmt.Println("\033[H\033[2J")
}

type csvLine struct {
	namespace string
	project   string
	branches  []string
}

type csvInputs []csvLine

func getInputFromStream(reader io.Reader) (csvInputs, error) {
	lines, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, err
	}
	csvInputs := csvInputs{}
	for index, line := range lines {
		if len(line) != 3 {
			return nil, fmt.Errorf("line %d is not valid", index+1)
		}
		if index == 0 {
			continue
		}
		csvLine := csvLine{line[0], line[1], strings.Fields(line[2])}
		csvInputs = append(csvInputs, csvLine)
	}
	return csvInputs, nil
}

func getInputFromHTTP(url string) (csvInputs, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return getInputFromStream(strings.NewReader(string(bytes)))
}

func getInputFromFile(path string) (csvInputs, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return getInputFromStream(file)
}

func getInput(path string) (csvInputs, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return getInputFromHTTP(path)
	} else {
		return getInputFromFile(strings.Replace(path, "file:/", "/", 1))
	}
}

func getInputFromEnv(env string) (csvInputs, error) {
	if value, found := os.LookupEnv(env); found {
		return getInputFromStream(strings.NewReader(value))
	}
	return csvInputs{}, nil
}

func getError(err error) *string {
	if err != nil {
		err := err.Error()
		return &err
	}
	return nil
}

func getProject(client *gitlab.Client, namespace, name string, branches ...string) Project {
	// get project
	project := func() Project {
		project, err := client.GetProject(namespace, name)
		return Project{Project: project, Branches: []Branch{}, Error: getError(err)}
	}()
	if project.Error != nil {
		return project
	}
	// get pipelines
	pipelinesMap, err := client.GetPipelines(project.ID, branches...)
	// propagate error to project
	project.Error = getError(err)
	if project.Error != nil {
		return project
	}
	// sort branches
	keys := make([]string, 0, len(pipelinesMap))
	for key := range pipelinesMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		pipelines := []Pipeline{}
		for _, pipeline := range pipelinesMap[key] {
			pipelineURL := strings.Join([]string{project.URL, "pipelines", strconv.Itoa(pipeline.ID)}, "/")
			pipelines = append(pipelines, Pipeline{pipeline, pipelineURL})
		}
		branch := Branch{Name: key, Pipelines: pipelines}
		project.Branches = append(project.Branches, branch)
	}
	return project
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		value, err := strconv.Atoi(value)
		if err == nil {
			return value
		}
	}
	return fallback
}

func main() {
	var protocol, host, token, addr string
	var api, interval, history, count int
	flag.Usage = func() {
		printErrf("Usage: %s [OPTIONS] [csv]\n\n", bin)
		printErrf("\t%s is a simple dashboard for your pipelines running in GitLab\n", bin)
		printErrln()
		printErrf("Options:\n")
		printErrf("\t-%s=<token>        gitlab access token (env %s, mandatory)\n", gitlabTokenProperty, envGitlabToken)
		printErrf("\t-%s=<protcol>      gitlab protocol (env %s, default %s)\n", gitlabProtocolProperty, envGitlabProtocol, gitlabProtocolDefault)
		printErrf("\t-%s=<host>         gitlab host (env %s, default %s)\n", gitlabHostProperty, envGitlabHost, gitlabHostDefault)
		printErrf("\t-%s=<version>      gitlab API version (env %s, default %d)\n", gitlabAPIProperty, envGitlabAPI, gitlabAPIDefault)
		printErrf("\t-%s=<seconds>      Refresh interval in seconds (env %s, default %d)\n", refreshIntervalProperty, envRefreshInterval, refreshIntervalDefault)
		printErrf("\t-%s=<addr>         Listen on given addr (env %s, default %s)\n", listenAddrProperty, envListenAddr, listenAddrDefault)
		printErrf("\t-%s=<count>        Stop after displaying <count> dashboards  (env %s, default %d)\n", countProperty, envCount, countDefault)
		printErrf("\t-%s=<history>      Pipeline history (env %s, default %d)\n", historyProperty, envHistory, historyDefault)
		printErrf("\t-h                 Display this help and exit\n")
		printErrln()
		printErrf("CSV:\n")
		printErrf("\tThere are two ways how to pass CSV to %s:\n", bin)
		printErrf("\t\t1. Argument(s)\n\t\t\t%s projects.csv\n", bin)
		printErrf("\t\t2. Set env %s\n\t\t\t%s=$(cat projects.csv) %s\n", envGitlabProjectsCSV, envGitlabProjectsCSV, bin)
		printErrln()
		printErrf("CSV example:\n")
		printErrf("\tnamespace,project,branches\n")
		printErrf("\t\"our-namespace\",\"cool-project\",\"master dev features/*\"\n")
		printErrf("\t\"my-namespace\",\"swag-project\",\"develop features/*\"\n")
		printErrf("\t\"your-namespace\",\"dead-project\",\"master\"\n")
		printErrln()
	}
	flag.StringVar(&token, gitlabTokenProperty, "", "")
	flag.StringVar(&protocol, gitlabProtocolProperty, getEnv(envGitlabProtocol, gitlabProtocolDefault), "")
	flag.StringVar(&host, gitlabHostProperty, getEnv(envGitlabHost, gitlabHostDefault), "")
	flag.IntVar(&api, gitlabAPIProperty, getEnvInt(envGitlabAPI, gitlabAPIDefault), "")
	flag.IntVar(&interval, refreshIntervalProperty, getEnvInt(envRefreshInterval, refreshIntervalDefault), "")
	flag.StringVar(&addr, listenAddrProperty, getEnv(envListenAddr, listenAddrDefault), "")
	flag.IntVar(&count, countProperty, getEnvInt(envCount, countDefault), "")
	flag.IntVar(&history, historyProperty, getEnvInt(envHistory, historyDefault), "")
	flag.Parse()
	if !isSet(token) {
		token = getEnv(envGitlabToken, "")
		if !isSet(token) {
			check(fmt.Errorf("token is not set (re-run with -h)"))
		}
	}

	csvInputs := csvInputs{}
	for _, arg := range flag.Args() {
		inputs, err := getInput(arg)
		check(err)
		csvInputs = append(csvInputs, inputs...)
	}
	inputs, err := getInputFromEnv(envGitlabProjectsCSV)
	check(err)
	csvInputs = append(csvInputs, inputs...)
	if len(csvInputs) == 0 {
		check(fmt.Errorf("missing input"))
	}
	client := gitlab.New(protocol, host, api, token)

	projects := []Project{}
	pProjects := &projects

	handler := func(w http.ResponseWriter, r *http.Request) {
		projects := *pProjects
		body, err := json.Marshal(projects)
		if err != nil {
			w.Write([]byte(err.Error()))
		}
		w.Write(body)
	}
	fs := http.FileServer(http.Dir("/web"))

	http.Handle("/web/", http.StripPrefix("/web/", fs))
	http.HandleFunc("/", handler)
	
	go http.ListenAndServe(addr, nil)

	for _, csvInput := range csvInputs {
		project := getProject(client, csvInput.namespace, csvInput.project, csvInput.branches...)
		projects = append(projects, project)
	}
	pProjects = &projects
	for i := 0; i < 10; i++ {
		clear()
		Display(projects, history)
		//Dump(projects)

		projects = []Project{}
		for _, csvInput := range csvInputs {
			project := getProject(client, csvInput.namespace, csvInput.project, csvInput.branches...)
			projects = append(projects, project)
		}
		pProjects = &projects
		<-time.After(time.Second * time.Duration(interval))
	}
}
