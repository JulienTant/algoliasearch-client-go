package it

import (
	"testing"

	"github.com/algolia/algoliasearch-client-go/algoliasearch"
	"github.com/stretchr/testify/require"
)

func TestSynonym(t *testing.T) {
	t.Parallel()
	_, index, _ := initClientAndIndex(t)

	var taskIDs []int

	{
		res, err := index.AddObjects([]algoliasearch.Object{
			{"console": "Sony PlayStation <PLAYSTATIONVERSION>"},
			{"console": "Nintendo Switch"},
			{"console": "Nintendo Wii U"},
			{"console": "Nintendo Game Boy Advance"},
			{"console": "Microsoft Xbox"},
			{"console": "Microsoft Xbox 360"},
			{"console": "Microsoft Xbox One"},
		})
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	synonyms := []algoliasearch.Synonym{
		algoliasearch.NewSynonym("gba", []string{"gba", "gameboy advance", "game boy advance"}),
		algoliasearch.NewOneWaySynonym("wii_to_wii_u", "Wii", []string{"Wii U"}),
		algoliasearch.NewPlaceholderSynonym("playstation_version_placeholder", "<PLAYSTATIONVERSION>", []string{"1", "One", "2", "3", "4", "4 Pro"}),
		algoliasearch.NewAltCorrectionSynonym("psone", []string{"robpike"}, "rob", algoliasearch.AltCorrection1),
		algoliasearch.NewAltCorrectionSynonym("pike", []string{"robpike"}, "pike", algoliasearch.AltCorrection2),
	}

	{
		res, err := index.SaveSynonym(synonyms[0], false)
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	{
		res, err := index.BatchSynonyms(synonyms[1:], false, false)
		require.NoError(t, err)
		taskIDs = append(taskIDs, res.TaskID)
	}

	waitTasks(t, index, taskIDs...)
	taskIDs = []int{}

	{
		for _, synonym := range synonyms {
			expected := synonym
			objectID := synonym.ObjectID
			t.Run("GetSynonym_"+expected.ObjectID, func(t *testing.T) {
				t.Parallel()
				found, err := index.GetSynonym(objectID)
				require.NoError(t, err)
				require.Equal(t, expected, found)
			})
		}
	}

	{
		t.Run("SearchSynonym", func(t *testing.T) {
			t.Parallel()
			found, err := index.SearchSynonyms("", nil, 0, 10)
			require.NoError(t, err)
			require.Equal(t, 5, len(found))
		})
	}

}
