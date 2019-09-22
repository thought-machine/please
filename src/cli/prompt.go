package cli

import (
	"strings"

	"github.com/manifoldco/promptui"
)

// PromptYN shows a yes/no prompt for the given question.
// It returns true if the answer was affirmative.
func PromptYN(msg string, defaultYes bool) bool {
	prompt := promptui.Prompt{
		Label:     msg,
		IsConfirm: true,
		Default:   "N",
	}
	if defaultYes {
		prompt.Default = "Y"
	}
	input, err := prompt.Run()
	if err != nil {
		if err.Error() == "" {
			return defaultYes // Happens when the user enters nothing
		}
		return false // most likely ctrl+C etc
	}
	return strings.ToLower(input) == "y" || (input == "" && defaultYes)
}
