package cli

import (
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
	_, err := prompt.Run()

	if err == promptui.ErrInterrupt {
		return false
	}
	// ErrAbort is returned when the user enters n (or the default value is n)
	return err != promptui.ErrAbort
}

func Prompt(msg string, defaultVal string) (string, error) {
	prompt := promptui.Prompt{
		Label:   msg,
		Default: defaultVal,
	}

	result, err := prompt.Run()
	if err != nil {
		if err.Error() == "" {
			return defaultVal, nil // Happens when the user enters nothing
		}
		return "", err
	}

	return result, nil
}
