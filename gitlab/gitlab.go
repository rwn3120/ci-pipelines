package gitlab

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Gitlab is a client
type Client struct {
	endpoint       string
	token          string
	client         *http.Client
	cachedProjects map[string]Project
}

// New creates a new gitlab client
func New(protocol string, host string, apiVersion int, token string) *Client {
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true}}
	return &Client{
		fmt.Sprintf("%s://%s/api/v%d", protocol, host, apiVersion),
		token,
		&http.Client{Transport: tr},
		map[string]Project{}}
}

func (c *Client) get(path string, args ...string) (*http.Response, error) {
	URL := c.endpoint + "/" + path
	arguments := strings.Join(args, "&")
	if len(arguments) > 0 {
		URL = URL + "?" + arguments
	}
	// fmt.Println("URL:", URL)
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Private-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		error := Error{}
		err = json.Unmarshal([]byte(contents), &error)
		if err != nil {
			return nil, fmt.Errorf("%d %s", resp.StatusCode, contents)
		}
		return nil, fmt.Errorf("%s", error.Message)
	}
	return resp, nil
}

// ListProjects returns list of projects
func (c *Client) ListProjects() ([]Project, error) {
	resp, err := c.get("projects", "per_page=10000")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	projects := []Project{}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject returns project
func (c *Client) GetProject(namespace, name string) (Project, error) {
	unknownProject := func(namespace, name string) Project {
		return Project{-1, name, "", "", Namespace{-1, namespace}}
	}
	nameWithNamespace := nameWithNamespace(namespace, name)
	cachedProject, found := c.cachedProjects[nameWithNamespace]
	if !found {
		projects, err := c.ListProjects()
		if err != nil {
			return unknownProject(namespace, name), err
		}
		c.cachedProjects = map[string]Project{}
		for _, project := range projects {
			c.cachedProjects[project.NameWithNamespace()] = project
		}
		fmt.Println("Cached", len(c.cachedProjects), "projects")
		cachedProject, found = c.cachedProjects[nameWithNamespace]
		if !found {
			return unknownProject(namespace, name), fmt.Errorf("Project %s does not exists", nameWithNamespace)
		}
	}
	return cachedProject, nil
}

// GetPipelines returns pipelines for given project
func (c *Client) GetPipelines(projectID int, branches ...string) (map[string][]Pipeline, error) {
	filteredPipelines := map[string][]Pipeline{}
	resp, err := c.get(fmt.Sprintf("projects/%d/pipelines", projectID), "per_page=10000")
	if err != nil {
		return filteredPipelines, err
	}
	defer resp.Body.Close()
	pipelines := []Pipeline{}
	err = json.NewDecoder(resp.Body).Decode(&pipelines)
	if err != nil {
		return filteredPipelines, err
	}

	for _, branch := range branches {
		if _, found := filteredPipelines[branch]; !found && !strings.Contains(branch, "*") {
			filteredPipelines[branch] = []Pipeline{}
		}
	}
	for _, pipeline := range pipelines {
		for _, branch := range branches {
			// if strings.Compare(branch, pipeline.Branch) == 0 {
			if strings.Contains(branch, "*") {
				regexp := regexp.MustCompile(strings.Replace(branch, "*", ".*", -1))
				if regexp.MatchString(pipeline.Branch) {
					filteredPipelines[pipeline.Branch] = append(filteredPipelines[branch], pipeline)
					break
				}
			} else {
				if strings.Compare(branch, pipeline.Branch) == 0 {
					filteredPipelines[branch] = append(filteredPipelines[branch], pipeline)
					break
				}
			}
		}
	}
	return filteredPipelines, err
}
