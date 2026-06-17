package tui

import (
	"encoding/json"
	"os"

	"github.com/charles-albert-raymond/synco/internal/state"
	"github.com/charles-albert-raymond/synco/internal/worktree"
)

type createIntentMsg struct {
	Branch          string                `json:"branch"`
	Title           string                `json:"title"`
	Base            string                `json:"base"`
	Source          worktree.BranchSource `json:"source"`
	UseSource       bool                  `json:"use_source"`
	CreationProfile string                `json:"creation_profile"`
}

type deleteIntentMsg struct {
	Entry        state.Entry `json:"entry"`
	DeleteBranch bool        `json:"delete_branch"`
}

type editTitleIntentMsg struct {
	Branch string `json:"branch"`
	Title  string `json:"title"`
}

type popupIntentEnvelope struct {
	Kind      string              `json:"kind"`
	Create    *createIntentMsg    `json:"create,omitempty"`
	Delete    *deleteIntentMsg    `json:"delete,omitempty"`
	EditTitle *editTitleIntentMsg `json:"edit_title,omitempty"`
}

func writePopupIntent(path string, env popupIntentEnvelope) error {
	if path == "" {
		return nil
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func readPopupIntent(path string) (popupIntentEnvelope, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return popupIntentEnvelope{}, false, nil
	}
	if err != nil {
		return popupIntentEnvelope{}, false, err
	}
	if len(data) == 0 {
		return popupIntentEnvelope{}, false, nil
	}
	var env popupIntentEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return popupIntentEnvelope{}, false, err
	}
	return env, true, nil
}
