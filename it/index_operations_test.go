package it

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/algolia/algoliasearch-client-go/algoliasearch"
)

func TestIndexOperations(t *testing.T) {
	t.Parallel()
	client, index, indexName := initClientAndIndex(t)

	var taskIDs []int

	{
		res, err := index.AddObjects([]algoliasearch.Object{
			{"objectID": "one", "company": "apple"},
			{"objectID": "two", "company": "tesla"},
		})
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	expectedSettings := algoliasearch.Map{
		"attributesForFaceting": []string{"company"},
	}

	{
		res, err := index.SetSettings(expectedSettings)
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	expectedSynonym := algoliasearch.NewPlaceholderSynonym(
		"google_placeholder",
		"<GOOG>",
		[]string{"Google", "GOOG"},
	)

	{
		res, err := index.SaveSynonym(expectedSynonym, true)
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	expectedRule := algoliasearch.Rule{
		ObjectID:  "company_auto_faceting",
		Condition: algoliasearch.NewSimpleRuleCondition(
			algoliasearch.Contains,
			"{facet:company}",
		),
		Consequence: algoliasearch.RuleConsequence{
			Params: algoliasearch.Map{
				"automaticFacetFilters": []string{"company"},
			},
		},
	}

	{
		res, err := index.SaveRule(expectedRule, true)
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	waitTasks(t, index, taskIDs...)
	taskIDs = []int{}

	{
		res, err := index.ScopedCopy(indexName+"_settings", []string{"settings"})
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	{
		res, err := index.ScopedCopy(indexName+"_rules", []string{"rules"})
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	{
		res, err := index.ScopedCopy(indexName+"_synonyms", []string{"synonyms"})
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	{
		res, err := index.Copy(indexName+"_copy")
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	waitTasks(t, index, taskIDs...)
	taskIDs = []int{}

	for _, c := range []struct {
		IndexName string
		ShouldHaveSameSettings bool
		ShouldHaveSameRules bool
		ShouldHaveSameSynonyms bool
	} {
		{IndexName: indexName+"_settings", ShouldHaveSameSettings: true},
		{IndexName: indexName+"_rules", ShouldHaveSameRules: true},
		{IndexName: indexName+"_synonyms", ShouldHaveSameSynonyms: true},
		{IndexName: indexName+"_copy", ShouldHaveSameSettings: true, ShouldHaveSameRules: true, ShouldHaveSameSynonyms: true},
	} {
		copiedIndex := client.InitIndex(c.IndexName)

		if c.ShouldHaveSameSettings {
			settings, err := copiedIndex.GetSettings()
			require.NoError(t, err)
			require.Equal(t, expectedSettings["attributesForFaceting"], settings.AttributesForFaceting)
		}

		if c.ShouldHaveSameRules {
			_, err := copiedIndex.GetRule(expectedRule.ObjectID)
			require.NoError(t, err)
		}

		if c.ShouldHaveSameSynonyms {
			_, err := copiedIndex.GetSynonym(expectedSynonym.ObjectID)
			require.NoError(t, err)
		}
	}
}
