package scm

import "fmt"

type stub struct{}

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
