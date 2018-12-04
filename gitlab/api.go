package gitlab

import (
	"fmt"
)

type Error struct {
	Message string `json:"error"`
}

type Namespace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"Description"`
	URL         string    `json:"web_url"`
	Namespace   Namespace `json:"namespace"`
}

func nameWithNamespace(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func (p Project) NameWithNamespace() string {
	return nameWithNamespace(p.Namespace.Name, p.Name)
}

func (p Project) String() string {
	return fmt.Sprintf("%s (ID %d)", p.NameWithNamespace(), p.ID)
}

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Username string `json:"username"`
	URL      string `json:"web_url"`
	Avatar   string `json:"avatar_url"`
}

type Pipeline struct {
	ID          int     `json:"id"`
	Status      string  `json:"status"`
	Branch      string  `json:"ref"`
	CreatedAt   string  `json:"created_at"`
	UpdateAt    string  `json:"updated_at"`
	StartedAt   string  `json:"started_at"`
	FinishedAt  string  `json:"finished_at"`
	CommittedAt string  `json:"committed_at"`
	Duration    float64 `json:"duration"`
	User        User    `json:"user`
}

func (p Pipeline) String() string {
	return fmt.Sprintf("%d %s (%s)", p.ID, p.Status, p.Branch)
}
