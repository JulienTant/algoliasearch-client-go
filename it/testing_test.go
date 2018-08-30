package it

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/algolia/algoliasearch-client-go/algoliasearch"
	"github.com/stretchr/testify/require"
)

func addDummyObject(t *testing.T, index algoliasearch.Index) {
	res, err := index.AddObject(algoliasearch.Object{"objectID": "one", "attribute": "value"})
	require.NoError(t, err)
	waitTasks(t, index, res.TaskID)
}

func initClientAndIndex(t *testing.T) (algoliasearch.Client, algoliasearch.Index, string) {
	c := initClient(t)
	instanceName := determineInstanceName()
	indexName := fmt.Sprintf(
		"%s_%s_%s",
		time.Now().Format("2006_01_02_15_04_05"),
		instanceName,
		t.Name(),
	)
	indexName = strings.Replace(indexName, "/", "_", -1)
	indexName = strings.Replace(indexName, " ", "_", -1)
	i := c.InitIndex(indexName)
	go deleteOldIndices(c)
	return c, i, indexName
}

func initClient(t *testing.T) algoliasearch.Client {
	appID := os.Getenv("ALGOLIA_APPLICATION_ID")
	key := os.Getenv("ALGOLIA_ADMIN_API_KEY")
	require.NotEmpty(t, appID)
	require.NotEmpty(t, key)
	return algoliasearch.NewClient(appID, key)
}

func determineInstanceName() string {
	if build := os.Getenv("TRAVIS_BUILD_NUMBER"); build != "" {
		return build
	}
	if user, err := user.Current(); err == nil {
		return user.Username
	}
	return "unknown"
}

func deleteOldIndices(c algoliasearch.Client) {
	today := time.Now().Format("2006_01_02")
	indices, _ := c.ListIndexes()
	for _, index := range indices {
		if !strings.HasPrefix(index.Name, today) {
			c.InitIndex(index.Name).Delete()
		}
	}
}

func waitTasks(t *testing.T, i algoliasearch.Index, tasks ...int) {
	var wg sync.WaitGroup
	for _, taskID := range tasks {
		wg.Add(1)
		go func(wg *sync.WaitGroup, taskID int) {
			defer wg.Done()
			err := i.WaitTask(taskID)
			require.NoError(t, err)
		}(&wg, taskID)
	}
	wg.Wait()
}

func getObjectAndCompareWith(t *testing.T, wg *sync.WaitGroup, index algoliasearch.Index, expected algoliasearch.Object) {
	defer wg.Done()
	objectID, err := expected.ObjectID()
	require.NoError(t, err)
	object, err := index.GetObject(objectID, nil)
	require.NoError(t, err)
	require.Equal(t, expected, object)
}

func getObjectsAndCompareWith(t *testing.T, wg *sync.WaitGroup, index algoliasearch.Index, objectIDs []string, expected []algoliasearch.Object) {
	defer wg.Done()
	objects, err := index.GetObjects(objectIDs)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, objects)
}
