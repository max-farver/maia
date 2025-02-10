package codecov

import (
	"fmt"
	"strings"

	gitm "github.com/aymanbagabas/git-module"
	"github.com/max-farver/maia/internal/git"
	"golang.org/x/tools/cover"
)

func GetDiff() ([]*gitm.DiffFile, error) {
	repo, err := git.GetRepo(".")
	if err != nil {
		return nil, fmt.Errorf("error getting repo: %v", err)
	}
	diff, err := git.GetDiff(repo, repo.HeadBranchName)
	if err != nil {
		return nil, fmt.Errorf("error getting diff: %v", err)
	}

	return diff, nil
}

func GetCoverage(diff []*gitm.DiffFile, coverageFile string) (float64, error) {
	if len(diff) == 0 {
		return 100.0, nil
	}

	diffMap := map[string][]*gitm.DiffLine{}

	totalLines := 0
	coveredLines := 0

	for _, file := range diff {
		for _, line := range file.Sections {
			diffMap[file.Name] = append(diffMap[file.Name], line.Lines...)
		}
	}

	coverageProfiles, err := cover.ParseProfiles(coverageFile)
	if err != nil {
		return 0.0, fmt.Errorf("error parsing coverage file: %v", err)
	}

	coverageMap := map[string]map[int]int{}
	for _, profile := range coverageProfiles {
		var fileName string
		parsedFileName := strings.SplitN(profile.FileName, "/", 4)
		if len(parsedFileName) == 4 {
			fileName = parsedFileName[3]
		} else {
			fileName = parsedFileName[2]
		}

		blocks := map[int]int{}
		for _, block := range profile.Blocks {
			blocks[block.StartLine] = block.Count
		}

		coverageMap[fileName] = blocks
	}

	isLineCovered := map[string]map[int]bool{}

	for diffFile, lines := range diffMap {
		for _, line := range lines {
			if !shouldCountCoverageForFile(diffFile) {
				continue
			}

			if isLineCovered[diffFile] == nil {
				isLineCovered[diffFile] = map[int]bool{}
			}

			if line.Type != gitm.DiffLineAdd && line.Type != gitm.DiffLinePlain {
				continue
			}

			totalLines++

			if coverage, ok := coverageMap[diffFile]; ok {
				if coverage[line.RightLine] > 0 {
					isLineCovered[diffFile][line.RightLine] = true
					coveredLines++
				} else {
					isLineCovered[diffFile][line.RightLine] = false
				}
			} else {
				isLineCovered[diffFile][line.RightLine] = false
			}
		}
	}

	if totalLines == 0 {
		return 0.0, nil
	}

	return float64(coveredLines) / float64(totalLines) * 100, nil
}

func shouldCountCoverageForFile(fileName string) bool {
	return strings.HasSuffix(fileName, ".go")
}
