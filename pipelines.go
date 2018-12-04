package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/rwn3120/ci-pipelines/gitlab"
)

const (
	resetStyle = "\033[0m"
)

type Branch struct {
	Name      string            `json:"branch"`
	Pipelines []gitlab.Pipeline `json:"pipelines"`
}

type Project struct {
	gitlab.Project
	Branches []Branch `json:"branches"`
	Error    *string  `json:"error"`
}

type coloredStatus struct {
	Value string
	Label string
	Style string
}

func (s coloredStatus) String() string {
	return s.Style + s.Label + resetStyle
}

var coloredStatuses = []coloredStatus{
	coloredStatus{"success", " S ", "\033[42m\033[1m"},
	coloredStatus{"failed", " F ", "\033[41m\033[1m"},
	coloredStatus{"running", " R ", "\033[44m"},
	coloredStatus{"pending", " P ", "\033[43m\033[30m"},
	coloredStatus{"canceled", " C ", "\033[40m"},
	coloredStatus{"skipped", " s ", "\033[100m"}}
var unknownStatus = coloredStatus{"unknown", " U ", "\033[40m\033[91m"}

var coloredBranches = map[string]string{
	"master":  "\033[93m\033[1m",
	"release": "\033[93m\033[1m",

	"dev":     "\033[94m\033[1m",
	"develop": "\033[94m\033[1m"}

func colorizeStatus(value string) coloredStatus {
	for _, status := range coloredStatuses {
		if strings.EqualFold(value, status.Value) {
			return status
		}
	}
	return unknownStatus
}

func colorizeBranch(value string) string {
	if len(value) > 16 {
		value = fmt.Sprintf("%s...", value[:16])
	}
	if style, found := coloredBranches[value]; found {
		return fmt.Sprintf("%s%s%s", style, value, resetStyle)
	}
	return fmt.Sprintf("%s%s%s", "\033[39m\033[0m", value, resetStyle)
}

func Dump(projects []Project) {
	for _, project := range projects {
		fmt.Printf("%s\n", project.Project)
		for _, branch := range project.Branches {
			fmt.Printf("\t%s\n", branch.Name)
			if len(branch.Pipelines) > 0 {
				for _, pipeline := range branch.Pipelines {
					fmt.Printf("\t\t%s\n", pipeline)
				}
			} else {
				fmt.Printf("\t\t<no pipelines>\n")
			}
		}
	}
}

func Display(projects []Project, history int) {
	// display legend
	fmt.Println("Legend:")
	for _, status := range coloredStatuses {
		fmt.Printf("\t%-10s %s\n", status.Value, status)
	}
	fmt.Println("")

	for _, project := range projects {
		if project.Error != nil {
			fmt.Printf("%-24s", project.Project.Name)
			fmt.Printf("\033[41m%s\033[0m\n", *project.Error)
			continue
		}
		for _, branch := range project.Branches {
			fmt.Printf("%-48s%32s  ", project.Project.NameWithNamespace(), colorizeBranch(branch.Name))
			if len(branch.Pipelines) > 0 {
				for index, pipeline := range branch.Pipelines {
					if index == 0 {
						fmt.Printf("%s%24s\t", colorizeStatus(pipeline.Status), pipeline.User.Name)
					} else {
						fmt.Printf("%s", colorizeStatus(pipeline.Status))
					}
					index++
					if index > history {
						break
					}
				}
			} else {
				fmt.Printf("%s%24s\t<no pipelines>", unknownStatus, "-")
			}
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n\nUpdated: %v\n", time.Now().Format(time.Stamp))
}
