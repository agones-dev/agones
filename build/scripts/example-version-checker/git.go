package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const cloneDir = "tmp"
const rootDir = "../../../"
const gitUrl = "https://github.com/agones-dev/agones.git"
const targetBranch = "refs/heads/main"

func getFailedFiles() []string {
	local, remote := getCommits()
	changed := getChangedFilenames(local, remote)
	sameVersionNames := getSameVersionExampleNames(local, remote)

	return filterFailedFiles(changed, sameVersionNames)
}

func filterFailedFiles(filePaths []string, sameVersionNames []string) []string {
	failed := make([]string, 0)

	for _, filePath := range filePaths {
		if filenameInExamples(filePath, sameVersionNames) {
			failed = append(failed, filePath)
		}
	}

	return failed
}

func getChangedFilenames(local *object.Commit, remote *object.Commit) []string {
	changes := getChanges(local, remote)
	exampleNames := getAllExampleNames()

	filenames := make([]string, 0)
	for _, change := range changes {
		filename := change.To.Name
		if !filenameIsIrrelevant(filename, exampleNames) {
			filenames = append(filenames, filename)
		}
	}

	return filenames
}

func getLocalRepo() *git.Repository {
	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		log.Fatalf("Failed to open local git repository: %v", err)
	}

	return repo
}

func getHeadCommit(repo *git.Repository) *object.Commit {
	ref, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		log.Fatalf("Failed to get HEAD reference: %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		log.Fatalf("Failed to get HEAD commit object: %v", err)
	}

	return commit
}

func getCommits() (*object.Commit, *object.Commit) {
	localRepo := getLocalRepo()
	localCommit := getHeadCommit(localRepo)

	remoteRepo := cloneRemoteRepo()
	remoteCommit := getHeadCommit(remoteRepo)

	return localCommit, remoteCommit
}

func getChanges(local *object.Commit, remote *object.Commit) object.Changes {
	localTree := getCommitTree(local)
	remoteTree := getCommitTree(remote)

	changes, err := object.DiffTree(localTree, remoteTree)
	if err != nil {
		log.Fatalf("Failed to diff trees: %v", err)
	}

	return changes
}

func getCommitTree(commit *object.Commit) *object.Tree {
	tree, err := commit.Tree()
	if err != nil {
		log.Fatalf("Failed to get tree: %v", err)
	}

	return tree
}

func exampleVersionChanged(exampleName string, local *object.Commit, remote *object.Commit) bool {
	log.Printf("Getting versions of %s", exampleName)

	localVersion, errLocal := getVersionFromCommit(exampleName, local)
	if errLocal != nil {
		log.Fatalf("Could not get (local) version of example %s: %v", exampleName, errLocal)
	}
	log.Printf("\tLocal:\t\t%s", localVersion)

	remoteVersion, errRemote := getVersionFromCommit(exampleName, remote)
	if errRemote != nil {
		return true
	}
	log.Printf("\tUpstream:\t%s", remoteVersion)

	return localVersion != remoteVersion
}

func getVersionFromCommit(exampleName string, commit *object.Commit) (string, error) {
	filePath := fmt.Sprintf("%s/%s/Makefile", examplesDir, exampleName)

	contents, err := getFileContents(commit, filePath)
	if err != nil {
		return "", err
	}

	return getVersionFromMakefile(contents)
}

func getFileContents(commit *object.Commit, filePath string) (string, error) {
	tree := getCommitTree(commit)

	return getFileContentsFromTree(tree, filePath)
}

func getFileContentsFromTree(tree *object.Tree, filePath string) (string, error) {
	file, err := tree.File(filePath)
	if err != nil {
		return "", err
	}

	reader, err := file.Reader()
	if err != nil {
		return "", err
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func cloneRemoteRepo() *git.Repository {
	os.RemoveAll(cloneDir)
	cloneOptions := &git.CloneOptions{
		URL:           gitUrl,
		ReferenceName: targetBranch,
	}
	repo, err := git.PlainClone(cloneDir, false, cloneOptions)
	if err != nil {
		log.Fatalf("Failed to clone the upstream git repository: %v", err)
	}
	return repo
}

func getSameVersionExampleNames(local *object.Commit, remote *object.Commit) []string {
	exampleNames := getAllExampleNames()

	sameVersionNames := make([]string, 0)
	for _, exampleName := range exampleNames {
		if !exampleVersionChanged(exampleName, local, remote) {
			sameVersionNames = append(sameVersionNames, exampleName)
		}
	}

	return sameVersionNames
}
