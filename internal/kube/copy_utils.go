package kube

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/max-farver/maia/internal/kube/utils"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type model struct {
	pod         string
	container   string
	fileToCopy  fileNameAccessor
	destination string
	isConfirmed bool
}

func (m *model) GetDestination() string {
	if m.destination == "" {
		return m.fileToCopy.name
	}
	return m.destination
}

type fileNameAccessor struct {
	fullPath string
	name     string
}

func (a *fileNameAccessor) Get() string {
	return a.name
}

func (a *fileNameAccessor) Set(fullPath string) {
	a.fullPath = fullPath
	a.name = filepath.Base(fullPath)
}

var CopyToPodCmd = &cobra.Command{
	Use:   "copy-to-pod",
	Short: "Copy a file or directory to a kubernetes pod.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		model := model{}
		form, _ := model.getForm()
		form.Run()

		fmt.Println(
			lipgloss.NewStyle().
				Width(40).
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("63")).
				Padding(1, 2).
				Render(fmt.Sprintf("%v", model)),
		)
	},
}

func (m *model) getForm() (*huh.Form, error) {
	cwd, _ := os.Getwd()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which pod would you like to copy to?").
				Options(huh.NewOptions(utils.ListActivePods()...)...).
				Value(&m.pod),

			huh.NewSelect[string]().
				Title("Which container would you like to copy to?").
				OptionsFunc(func() []huh.Option[string] {
					return huh.NewOptions(utils.ListContainersForPod(m.pod)...)
				}, &m.pod).
				Value(&m.container),
		),

		huh.NewGroup(
			huh.NewFilePicker().
				Title("Which file would you like to copy?").
				CurrentDirectory(cwd).
				Accessor(&m.fileToCopy),

			huh.NewInput().
				Title("What should the output destination and file name be?").
				PlaceholderFunc(func() string {
					m.destination = m.fileToCopy.name
					return fmt.Sprintf("/home/%s", m.fileToCopy.name)
				}, &m.fileToCopy.name).
				Value(&m.destination),
		),

		// Gather some final details about the order.
		huh.NewGroup(
			huh.NewConfirm().
				TitleFunc(func() string {

					return fmt.Sprintf("Are you sure you'd like to copy %[1]s to container %[2]s in pod %[3]s in location %[4]s?", m.fileToCopy.Get(), m.container, m.pod, m.GetDestination())
				}, &m).
				Value(&m.isConfirmed),
		),
	)

	return form, nil
}
