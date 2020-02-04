package scm

import "fmt"

type stub struct{}

func (s *stub) DescribeIdentifier(sha string) string {
	return "<unknown>"
}

func (s *stub) CurrentRevIdentifier() string {
	return "<unknown>"
}

func (s *stub) ChangesIn(diffSpec string, relativeTo string) []string {
	return nil
}

func (s *stub) ChangedFiles(fromCommit string, includeUntracked bool, relativeTo string) []string {
	return nil
}

func (s *stub) IgnoreFile(name string) error {
	return fmt.Errorf("Don't know how to mark %s as ignored", name)
}

func (s *stub) Remove(names []string) error {
	return fmt.Errorf("Unknown SCM, can't remove files")
}

func (s *stub) ChangedLines() (map[string][]int, error) {
	return nil, fmt.Errorf("Unknown SCM, can't calculate changed lines")
}

func (s *stub) Checkout(revision string) error {
	return fmt.Errorf("Unknown SCM, can't checkout")
}

func (s *stub) CurrentRevDate(format string) string {
	return "Unknown"
}
