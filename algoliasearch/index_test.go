package algoliasearch

import (
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

func TestScopedCopy(t *testing.T) {
	t.Parallel()
	c, i := initClientAndIndex(t, "TestScopedCopy")
	iCopy := initIndex(t, c, "TestScopedCopy_copy")

	t.Log("TestScopedCopy: Add object, settings, synonyms and query rules")
	synonyms := addObjectsAndSynonyms(t, i, "TestScopedCopy") // Add objects, settings and synonyms
	addRules(t, i, "TestScopedCopy")                          // Add query rules

	t.Log("TestScopedCopy: Perform the scoped copy (settings and synonyms only)")
	{
		res, err := i.ScopedCopy("TestScopedCopy_copy", []string{"settings", "synonyms"})
		if err != nil {
			t.Fatalf("TestScopedCopy: Cannot scoped-copy the index: %s", err)
		}
		waitTask(t, i, res.TaskID)
	}

	t.Log("TestScopedCopy: Test the copied settings")
	{
		settings, err := i.GetSettings()
		if err != nil {
			t.Fatalf("TestScopedCopy: Cannot get settings of TestScopedCopy index: %s", err)
		}
		settingsCopy, err := iCopy.GetSettings()
		if err != nil {
			t.Fatalf("TestScopedCopy: Cannot get settings of TestScopedCopy_copy index: %s", err)
		}
		settingsAreEqual(t, settings, settingsCopy)
	}

	t.Log("TestScopedCopy: Test the copied synonyms")
	{
		foundSynonyms, err := iCopy.SearchSynonyms("", []string{}, 0, 1000)
		if err != nil {
			t.Fatalf("TestScopedCopy: Could not find any synonym with '' query: %s", err)
		}

		if !synonymSlicesAreEqual(synonyms, foundSynonyms) {
			t.Fatalf("TestScopedCopy: Synonym slices are not equal:\n%v\n%v", synonyms, foundSynonyms)
		}
	}

	t.Log("TestScopedCopy: Test the uncopied rules")
	{
		res, err := iCopy.SearchRules(Map{"query": ""})
		if err != nil {
			t.Fatalf("TestScopedCopy: Could not search for query rules in ScopedCopy_copy index: %s", err)
		}
		if len(res.Hits) > 0 {
			t.Fatalf("TestScopedCopy: Query rules must not have been copied")
		}
	}
}

func TestIndexOperations(t *testing.T) {
	t.Parallel()
	c, i := initClientAndIndex(t, "TestIndexOperations")

	objectID := addOneObject(t, i)

	t.Log("TestIndexOperations: Test Copy")
	{
		res, err := i.Copy("TestIndexOperations_copy")
		if err != nil {
			t.Fatalf("TestIndexOperations: Cannot copy the index: %s", err)
		}

		waitTask(t, i, res.TaskID)
	}

	t.Log("TestIndexOperations: Test Move")
	i = c.InitIndex("TestIndexOperations_copy")
	{
		res, err := i.Move("TestIndexOperations_move")
		if err != nil {
			t.Fatalf("TestIndexOperations: Cannot move the index: %s", err)
		}

		waitTask(t, i, res.TaskID)
	}

	t.Log("TestIndexOperations: Test Clear")
	i = c.InitIndex("TestIndexOperations_move")
	{
		res, err := i.Clear()
		if err != nil {
			t.Fatalf("TestClear: Cannot clear the index: %s", err)
		}

		waitTask(t, i, res.TaskID)

		_, err = i.GetObject(objectID, nil)
		if err == nil || err.Error() != "{\"message\":\"ObjectID does not exist\",\"status\":404}\n" {
			t.Fatalf("TestIndexOperations: Object %s should be deleted after clear: %s", objectID, err)
		}
	}

	t.Log("TestIndexOperations: Test Delete")
	{
		_, err := i.Delete()
		if err != nil {
			t.Fatalf("TestIndexOperations: Cannot delete the moved index: %s", err)
		}
	}
}

// stringSlicesAreEqual returns `true` if the two slices are the same i.e. if
// they contain the same strings. It returns `false` otherwise. Slices are
// sorted before the comparison.
func stringSlicesAreEqual(s1, s2 []string) bool {
	sort.Strings(s1)
	sort.Strings(s2)

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

// intSlicesAreEqual returns `true` if the two slices are the same i.e. if
// they contain the same integers. It returns `false` otherwise. Slices are
// sorted before the comparison.
func intSlicesAreEqual(s1, s2 []int) bool {
	sort.Ints(s1)
	sort.Ints(s2)

	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}

	return true
}

// settingsAreEqualByComparable returns `true` if all the comparable fields of
// the given Settings are the same. It returns `false` otherwise.
func settingsAreEqualByComparable(s1, s2 Settings) bool {
	return s1.AllowCompressionOfIntegerArray == s2.AllowCompressionOfIntegerArray &&
		s1.AdvancedSyntax == s2.AdvancedSyntax &&
		s1.AllowTyposOnNumericTokens == s2.AllowTyposOnNumericTokens &&
		s1.AttributeForDistinct == s2.AttributeForDistinct &&
		s1.HighlightPostTag == s2.HighlightPostTag &&
		s1.HighlightPreTag == s2.HighlightPreTag &&
		s1.HitsPerPage == s2.HitsPerPage &&
		s1.IgnorePlurals == s2.IgnorePlurals &&
		s1.KeepDiacriticsOnCharacters == s2.KeepDiacriticsOnCharacters &&
		s1.MaxValuesPerFacet == s2.MaxValuesPerFacet &&
		s1.MinProximity == s2.MinProximity &&
		s1.MinWordSizefor1Typo == s2.MinWordSizefor1Typo &&
		s1.MinWordSizefor2Typos == s2.MinWordSizefor2Typos &&
		s1.QueryType == s2.QueryType &&
		s1.ReplaceSynonymsInHighlight == s2.ReplaceSynonymsInHighlight &&
		s1.SeparatorsToIndex == s2.SeparatorsToIndex &&
		s1.SnippetEllipsisText == s2.SnippetEllipsisText &&
		s1.TypoTolerance == s2.TypoTolerance
}

// settingsAreEqualByStringSlices returns `true` if all the string slices of
// the given Settings are the same. It returns `false` otherwise`.
func settingsAreEqualByStringSlices(s1, s2 Settings) bool {
	return stringSlicesAreEqual(s1.AttributesForFaceting, s2.AttributesForFaceting) &&
		stringSlicesAreEqual(s1.SearchableAttributes, s2.SearchableAttributes) &&
		stringSlicesAreEqual(s1.CustomRanking, s2.CustomRanking) &&
		stringSlicesAreEqual(s1.NumericAttributesForFiltering, s2.NumericAttributesForFiltering) &&
		stringSlicesAreEqual(s1.Ranking, s2.Ranking) &&
		stringSlicesAreEqual(s1.Replicas, s2.Replicas) &&
		stringSlicesAreEqual(s1.UnretrievableAttributes, s2.UnretrievableAttributes) &&
		stringSlicesAreEqual(s1.DisableTypoToleranceOnAttributes, s2.DisableTypoToleranceOnAttributes) &&
		stringSlicesAreEqual(s1.DisableTypoToleranceOnWords, s2.DisableTypoToleranceOnWords) &&
		stringSlicesAreEqual(s1.AttributesToHighlight, s2.AttributesToHighlight) &&
		stringSlicesAreEqual(s1.AttributesToRetrieve, s2.AttributesToRetrieve) &&
		stringSlicesAreEqual(s1.AttributesToSnippet, s2.AttributesToSnippet) &&
		stringSlicesAreEqual(s1.OptionalWords, s2.OptionalWords)
}

// convertInterfaceSliceToStringSlice converts the input interface{} slice into
// a string slice. This function is only needed internally by
// `settingsAreEqualByRemoveStopWords` because of the way the Settings are
// unmarshal from the JSON.
func convertInterfaceSliceToStringSlice(in []interface{}) (out []string) {
	for i := 0; i < len(in); i++ {
		out = append(out, in[i].(string))
	}

	return
}

// settingsAreEqualByRemoveStopWords checks that the `removeStopWords` fields
// of the given Settings are the same (the type can be either a []string or a
// bool).
func settingsAreEqualByRemoveStopWords(t *testing.T, s1, s2 Settings) {
	itf1 := s1.RemoveStopWords
	itf2 := s2.RemoveStopWords

	is1, ok1 := itf1.([]interface{})
	ss2, ok2 := itf2.([]string)
	if ok1 && ok2 {
		ss1 := convertInterfaceSliceToStringSlice(is1)
		if stringSlicesAreEqual(ss1, ss2) {
			return
		} else {
			t.Fatalf("settingsAreEqualByRemoveStopWords: RemoveStopWords fields are different: %v != %v\n", ss1, ss2)
		}
	} else if !ok1 && !ok2 {
		b1, ok1 := itf1.(bool)
		b2, ok2 := itf2.(bool)

		if ok1 && ok2 {
			if b1 == b2 {
				return
			} else {
				t.Fatalf("settingsAreEqualByRemoveStopWords: RemoveStopWords fields are different: %t != %t\n", b1, b2)
			}
		}
	}

	t.Fatalf("settingsAreEqualByRemoveStopWords: RemoveStopWords fields are not typed as []string or bool: %v != %v\n", itf1, itf2)
}

// settingsAreEqualByDistinct checks that the `distinct` fields of the given
// Settings are the same (the type can be either a int or a bool).
func settingsAreEqualByDistinct(t *testing.T, s1, s2 Settings) {
	itf1 := s1.Distinct
	itf2 := s2.Distinct

	f1, ok1 := itf1.(float64)
	i2, ok2 := itf2.(int)
	if ok1 && ok2 {
		i1 := int(f1)
		if i1 == i2 {
			return
		} else {
			t.Fatalf("settingsAreEqualByDistinct: Distinct fields are different: %d != %d\n", i1, i2)
		}
	} else if !ok1 && !ok2 {
		b1, ok1 := itf1.(bool)
		b2, ok2 := itf2.(bool)

		if ok1 && ok2 {
			if b1 == b2 {
				return
			} else {
				t.Fatalf("settingsAreEqualByDistinct: Distinct fields are different: %t != %t\n", b1, b2)
			}
		}
	}

	t.Fatalf("settingsAreEqualByDistinct: Distinct fields are not typed as int or bool: %v != %v\n", itf1, itf2)
}

// settingsAreEqual deeply checks that the two Settings are the same.
func settingsAreEqual(t *testing.T, s1, s2 Settings) {
	if !settingsAreEqualByComparable(s1, s2) {
		t.Fatalf("settingsAreEqual: Comparable fields are not equal:\n%#v\n%#v\n", s1, s2)
	}

	if !settingsAreEqualByStringSlices(s1, s2) {
		t.Fatalf("settingsAreEqual: String slice fields are not equal:\n%#v\n%#v\n", s1, s2)
	}

	settingsAreEqualByRemoveStopWords(t, s1, s2)
	settingsAreEqualByDistinct(t, s1, s2)
}

// setAndGetAndCompareSettings is a simple wrapper for succesive calls to
// `SetSettings`, `GetSettings` and `settingsAreEqual`.
func setAndGetAndCompareSettings(t *testing.T, i Index, expectedSettings Settings, mapSettings Map) {
	res, err := i.SetSettings(mapSettings)
	if err != nil {
		t.Fatalf("setAndGetAndCompareSettings: Cannot set settings: %s", err)
	}
	waitTask(t, i, res.TaskID)

	settings, err := i.GetSettings()
	if err != nil {
		t.Fatalf("setAndGetAndCompareSettings: Cannot get settings: %s", err)
	}

	settingsAreEqual(t, settings, expectedSettings)
}

// objectsAreEqual returns `true` if the two Objects are deeply equal.
func objectsAreEqual(o1, o2 Object) bool {
	for k, v := range o1 {
		if o2[k] != v {
			return false
		}
	}

	return true
}

// objectsAreEqual returns `true` if the two slices contains the exact same
// Objects. Slices don't need to be sorted.
func objectSlicesAreEqual(t *testing.T, s1, s2 []Object) {
	if len(s1) != len(s2) {
		t.Fatalf("objectSlicesAreEqual: Slices have not the same size: (%d,%d)", len(s1), len(s2))
	}

	var objectIDs []string

	for _, o1 := range s1 {
		id1 := o1["objectID"].(string)

		for _, o2 := range s2 {
			id2 := o2["objectID"].(string)
			if id1 == id2 {
				if objectsAreEqual(o1, o2) {
					objectIDs = append(objectIDs, id1)
				} else {
					t.Fatalf("objectSlicesAreEqual: Objects are not the same:\n%#v\n!=\n%#v\n", o1, o2)
				}
			}
		}
	}

	if len(objectIDs) != len(s1) {
		t.Fatalf("objectSlicesAreEqual: Slices does not contain the same objects:\n%#v\n!=\n%#v\n", s1, s2)
	}
}

// getAllRecords returns all the records from the given index.
func getAllRecords(t *testing.T, i Index) (records []Map) {
	// Initialize the iterator
	it, err := i.BrowseAll(nil)
	if err != nil {
		t.Fatalf("getAllRecords: BrowseAll has failed: %s", err)
	}

	// Iterate through all the records
	record, err := it.Next()
	for err == nil {
		records = append(records, record)
		record, err = it.Next()
	}

	// Check if BrowseAll has finished properly
	if err.Error() != "No more hits" {
		t.Fatalf("getAllRecords: BrowseAll iterations have failed: %s", err)
	}

	return
}

// waitIndexKey waits until the key has been properly added to the given index
// and if the given function, if not `nil`, returns `true`.
func waitIndexKey(t *testing.T, i Index, keyID string, f func(k Key) bool) {
	retries := 120

	for r := 0; r < retries; r++ {
		key, err := i.GetAPIKey(keyID)

		if err == nil && (f == nil || f(key)) {
			return
		}
		time.Sleep(1 * time.Second)
	}

	t.Fatalf("waitIndexKey: Key not found or function call failed")
}

// waitIndexKeysAsync waits until all the keys have been properly added to the
// given index and if the given function, if not `nil`, returns `true` for
// every key.
func waitIndexKeysAsync(t *testing.T, i Index, keyIDs []string, f func(k Key) bool) {
	var wg sync.WaitGroup

	for _, keyID := range keyIDs {
		wg.Add(1)

		go func(keyID string) {
			defer wg.Done()
			waitIndexKey(t, i, keyID, f)
		}(keyID)
	}

	wg.Wait()
}

// deleteIndexKey deletes the key for the given index.
func deleteIndexKey(t *testing.T, i Index, key string) {
	_, err := i.DeleteAPIKey(key)
	if err != nil {
		t.Fatalf("deleteIndexKey: Cannot delete key: %s", err)
	}
}

// deleteAllIndexKeys properly deletes all previous keys associated to the
// index.
func deleteAllIndexKeys(t *testing.T, i Index) {
	keys, err := i.ListKeys()

	if err != nil {
		t.Fatalf("deleteAllKeys: Cannot list the keys: %s", err)
	}

	for _, key := range keys {
		_, err = i.DeleteAPIKey(key.Value)
		if err != nil {
			t.Fatalf("deleteAllKeys: Cannot delete a key: %s", err)
		}
	}

	for len(keys) != 0 {
		keys, err = i.ListKeys()

		if err != nil {
			t.Fatalf("deleteAllKeys: Cannot list the keys: %s", err)
		}

		time.Sleep(1 * time.Second)
	}
}

func TestIndexKeys(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestIndexKeys")

	addOneObject(t, i)

	deleteAllIndexKeys(t, i)

	t.Log("TestIndexKeys: Check that no key was previously existing")
	{
		keys, err := i.ListKeys()

		if err != nil {
			t.Fatalf("TestIndexKeys: Cannot list the keys: %s", err)
		}

		if len(keys) != 0 {
			t.Fatalf("TestIndexKeys: Should return 0 keys instead of %d", len(keys))
		}
	}

	var searchKey, allRightsKey string

	t.Log("TestIndexKeys: Add a search key with parameters")
	{
		params := Map{
			"description":            "",
			"maxQueriesPerIPPerHour": 1000,
			"referers":               []string{},
			"queryParameters":        "typoTolerance=strict",
			"validity":               600,
			"maxHitsPerQuery":        1,
		}

		res, err := i.AddAPIKey([]string{"search"}, params)
		if err != nil {
			t.Fatalf("TestIndexKeys: Cannot create the search key: %s", err)
		}

		searchKey = res.Key
	}
	defer deleteIndexKey(t, i, searchKey)

	t.Log("TestIndexKeys: Add an all-permissions key")
	{
		acl := []string{
			"search",
			"browse",
			"addObject",
			"deleteObject",
			"deleteIndex",
			"settings",
			"editSettings",
			"analytics",
			"listIndexes",
		}

		res, err := i.AddAPIKey(acl, nil)
		if err != nil {
			t.Fatalf("TestIndexKeys: Cannot create the all-rights key: %s", err)
		}

		allRightsKey = res.Key
	}
	defer deleteIndexKey(t, i, allRightsKey)

	waitIndexKeysAsync(t, i, []string{searchKey, allRightsKey}, nil)

	t.Log("TestIndexKeys: Update search key description")
	{
		params := Map{"description": "Search-Only Key"}

		_, err := i.UpdateAPIKey(searchKey, params)
		if err != nil {
			t.Fatalf("TestIndexKeys: Cannot update search only key's description: %s", err)
		}

		waitIndexKey(t, i, searchKey, func(k Key) bool { return k.Description == "Search-Only Key" })
	}
}

func TestSettingsToMap(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestSettingsToMap")

	addOneObject(t, i)

	t.Log("TestSettingsToMap: Get the original settings")
	settingsBefore, err := i.GetSettings()
	if err != nil {
		t.Fatalf("TestSettingsToMap: Cannot retrieve the settings (before): %s", err)
	}

	t.Log("TestSettingsToMap: Set the settings by calling `ToMap` on the settings")
	res, err := i.SetSettings(settingsBefore.ToMap())
	if err != nil {
		t.Fatalf("TestSettingsToMap: Cannot set the settings: %s", err)
	}
	waitTask(t, i, res.TaskID)

	t.Log("TestSettingsToMap: Get the settings once again")
	settingsAfter, err := i.GetSettings()
	if err != nil {
		t.Fatalf("TestSettingsToMap: Cannot retrieve the settings (after): %s", err)
	}

	t.Log("TestSettingsToMap: Compare the settings")
	settingsAreEqual(t, settingsBefore, settingsAfter)
}

func TestSettingsToMap_allRequiredFieldsArePresent(t *testing.T) {
	var settings Settings

	m := settings.ToMap()

	s := reflect.ValueOf(&settings).Elem()
	tt := s.Type()

	for i := 0; i < s.NumField(); i++ {
		// Skip []string-type fields because those are discarded to avoid
		// sending useless data to the API.
		if tt.Field(i).Type.String() == "[]string" {
			continue
		}

		expectedSettingName := tt.Field(i).Name
		tmp := []rune(expectedSettingName)
		tmp[0] = unicode.ToLower(tmp[0])
		expectedSettingName = string(tmp)

		_, ok := m[expectedSettingName]
		require.True(t, ok, "should find '%s' setting in the result map", expectedSettingName)
	}
}

func facetHitSliceAreEqual(fs1, fs2 []FacetHit) bool {
	if len(fs1) != len(fs2) {
		return false
	}

	ok := 0

	for _, f1 := range fs1 {
		for _, f2 := range fs2 {
			if f1 == f2 {
				ok++
				break
			}
		}
	}

	return ok == len(fs1)
}

func TestSearchForFacetValues(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestSearchForFacetValues")

	var tasks []int

	t.Log("TestSearchForFacetValues: Add multiple objects at once")
	{
		objects := []Object{
			{"company": "Algolia", "name": "Julien Lemoine"},
			{"company": "Algolia", "name": "Nicolas Dessaigne"},
			{"company": "Amazon", "name": "Jeff Bezos"},
			{"company": "Apple", "name": "Steve Jobs"},
			{"company": "Apple", "name": "Steve Wozniak"},
			{"company": "Arista Networks", "name": "Jayshree Ullal"},
			{"company": "Google", "name": "Larry Page"},
			{"company": "Google", "name": "Rob Pike"},
			{"company": "Google", "name": "Sergueï Brin"},
			{"company": "Microsoft", "name": "Bill Gates"},
			{"company": "SpaceX", "name": "Elon Musk"},
			{"company": "Tesla", "name": "Elon Musk"},
			{"company": "Yahoo", "name": "Marissa Mayer"},
		}
		res, err := i.AddObjects(objects)
		if err != nil {
			t.Fatalf("TestSearchForFacetValues: Cannot add multiple objects: %s", err)
		}
		tasks = append(tasks, res.TaskID)
	}

	t.Log("TestSearchForFacetValues: Set settings")
	{
		res, err := i.SetSettings(Map{
			"attributesForFaceting": []string{"searchable(company)"},
		})
		if err != nil {
			t.Fatalf("TestSearchForFacetValues: Cannot set attributesForFaceting setting: %s", err)
		}
		tasks = append(tasks, res.TaskID)
	}

	t.Log("TestSearchForFacetValues: Wait for all the previous tasks to complete")
	waitTasksAsync(t, i, tasks)

	t.Log("TestSearchForFacetValues: Run queries")
	{
		expected := []FacetHit{
			{Value: "Algolia", Highlighted: "<em>A</em>lgolia", Count: 2},
			{Value: "Amazon", Highlighted: "<em>A</em>mazon", Count: 1},
			{Value: "Apple", Highlighted: "<em>A</em>pple", Count: 2},
			{Value: "Arista Networks", Highlighted: "<em>A</em>rista Networks", Count: 1},
		}

		res, err := i.SearchForFacetValues("company", "a", nil)
		if err != nil {
			t.Fatalf("TestSearchForFacetValues: Cannot SearchForFacetValues: %s", err)
		}

		if len(res.FacetHits) != 4 {
			t.Fatalf("TestSearchForFacetValues: Should return 4 facet hits instead of %d", len(res.FacetHits))
		}

		if !facetHitSliceAreEqual(res.FacetHits, expected) {
			t.Fatalf("TestSearchForFacetValues: FacetHit slices should be equal:\nExpected: %#v\nGot: %#v\n", res.FacetHits, expected)
		}

		// Check that `SearchFacet` is behaving in the exact same way as
		// `SearchForFacetValues` as it was kept for backward-compatibily.
		res2, err2 := i.SearchFacet("company", "a", nil)
		if err != err2 {
			t.Fatalf("TestSearchForFacetValues: SearchFacet and SearchForFacetValues aren't returing the same error:\nearchForFacetValues: %#v\nSearchForFacet: %#v\n", err, err2)
		}

		if !facetHitSliceAreEqual(res.FacetHits, res2.FacetHits) {
			t.Fatalf("TestSearchForFacetValues: SearchFacet and SearchForFacetValues aren't returing the same slices:\nearchForFacetValues: %#v\nSearchForFacet: %#v\n", res.FacetHits, res2.FacetHits)
		}
	}

	{
		params := Map{
			"typoTolerance": "false",
		}

		res, err := i.SearchForFacetValues("company", "aglolia", params)
		if err != nil {
			t.Fatalf("TestSearchForFacetValues: Cannot SearchForFacetValues: %s", err)
		}

		if len(res.FacetHits) != 0 {
			t.Fatalf("TestSearchForFacetValues: Should return 0 facet hits instead of %d\nGot: %#v\n", len(res.FacetHits), res.FacetHits)
		}

		// Check that `SearchFacet` is behaving in the exact same way as
		// `SearchForFacetValues` as it was kept for backward-compatibily.
		res2, err2 := i.SearchFacet("company", "aglolia", params)
		if err != err2 {
			t.Fatalf("TestSearchForFacetValues: SearchFacet and SearchForFacetValues aren't returing the same error:\nearchForFacetValues: %#v\nSearchForFacet: %#v\n", err, err2)
		}

		if !facetHitSliceAreEqual(res.FacetHits, res2.FacetHits) {
			t.Fatalf("TestSearchForFacetValues: SearchFacet and SearchForFacetValues aren't returing the same slices:\nearchForFacetValues: %#v\nSearchForFacet: %#v\n", res.FacetHits, res2.FacetHits)
		}
	}
}

func TestGeoSearchParameters(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestGeoSearchParameters")

	addOneObject(t, i)

	t.Log("TestGeoSearchParameters: Test valid parameters")
	{
		validParams := []Map{
			{"insideBoundingBox": "1.0,2.0,3.0,4.0"},
			{"insideBoundingBox": "1.0,2.0,3.0,4.0,5.0,6.0,7.0,8.0"},
			{"insidePolygon": "1.0,2.0,3.0,4.0,5.0,6.0"},
			{"insidePolygon": "[[1.0,2.0,3.0,4.0,5.0,6.0],[1.0,2.0,3.0,4.0,5.0,6.0]]"},
		}

		for _, params := range validParams {
			if _, err := i.Search("", params); err != nil {
				t.Errorf("TestGeoSearchParameters: Parameters %#v should not have raised an error but got `%s` instead",
					params,
					err,
				)
			}
		}
	}

	t.Log("TestGeoSearchParameters: Test invalid parameters")
	{
		cases := []struct {
			params      Map
			expectedErr error
		}{
			{Map{"insideBoundingBox": []string{"1.0,2.0,3.0,4.0"}}, invalidType("insideBoundingBox", "string or [][]float64")},
			{Map{"insidePolygon": []string{"1.0,2.0,3.0,4.0"}}, invalidType("insidePolygon", "string or [][]float64")},
		}

		for _, c := range cases {
			if _, err := i.Search("", c.params); err == nil || err.Error() != c.expectedErr.Error() {
				t.Errorf("TestGeoSearchParameters: Parameters %#v should have raised an error `%s` but got `%s` instead",
					c.params,
					c.expectedErr,
					err,
				)
			}
		}
	}
}

func TestBatchPartialUpdate(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestBatchPartialUpdate")

	var objectID string

	t.Log("Add the object that will get partially updated")
	{
		object := Object{
			"replace":         10,
			"increment":       10,
			"decrement":       10,
			"addInt":          []int{1, 2, 3},
			"addString":       []string{"1", "2", "3"},
			"removeInt":       []int{1, 2, 3},
			"removeString":    []string{"1", "2", "3"},
			"addUniqueInt":    []int{1, 2, 3},
			"addUniqueString": []string{"1", "2", "3"},
		}

		res, err := i.AddObject(object)
		if err != nil {
			t.Fatalf("TestBatchPartialUpdate: Cannot add an object: %s", err)
		}

		waitTask(t, i, res.TaskID)

		objectID = res.ObjectID
	}

	t.Log("Partially update all the fields via a Batch")
	{
		batchOps := []BatchOperation{
			{
				Action: "partialUpdateObject",
				Body: map[string]interface{}{
					"objectID":        objectID,
					"replace":         0,
					"increment":       IncrementOp(5),
					"decrement":       DecrementOp(5),
					"addInt":          AddOp(3),
					"addString":       AddOp("3"),
					"removeInt":       RemoveOp(3),
					"removeString":    RemoveOp("3"),
					"addUniqueInt":    AddUniqueOp(3),
					"addUniqueString": AddUniqueOp("3"),
				},
			},
		}

		res, err := i.Batch(batchOps)
		if err != nil {
			t.Fatalf("TestBatchPartialUpdate: Cannot batch the partial update operation: %s", err)
		}

		waitTask(t, i, res.TaskID)
	}

	t.Log("Check the final object")
	{
		object, err := i.GetObject(objectID, nil)

		if err != nil {
			t.Fatalf("TestBatchPartialUpdate: Cannot get the final object: %s", err)
		}

		{
			itf, ok := object["replace"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute replace missing")
			}

			value := int64(itf.(float64))
			if value != 0 {
				t.Errorf("TestBatchPartialUpdate: Wrong value for replace attribute, %d should be 0", value)
			}
		}

		{
			itf, ok := object["increment"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute increment missing")
			}

			value := int64(itf.(float64))
			if value != 15 {
				t.Errorf("TestBatchPartialUpdate: Wrong value for increment attribute, %d should be 15", value)
			}
		}

		{
			itf, ok := object["decrement"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute decrement missing")
			}

			value := int64(itf.(float64))
			if value != 5 {
				t.Errorf("TestBatchPartialUpdate: Wrong value for decrement attribute, %d should be 5", value)
			}
		}

		{
			itf, ok := object["addInt"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute addInt missing")
			}

			sitf := itf.([]interface{})
			s := make([]int, len(sitf))
			for i := range sitf {
				s[i] = int(sitf[i].(float64))
			}

			if !intSlicesAreEqual(s, []int{1, 2, 3, 3}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for addInt attribute, %v should be []int{1, 2, 3, 3}", s)
			}
		}

		{
			itf, ok := object["addString"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute addString missing")
			}

			sitf := itf.([]interface{})
			s := make([]string, len(sitf))
			for i := range sitf {
				s[i] = sitf[i].(string)
			}

			if !stringSlicesAreEqual(s, []string{"1", "2", "3", "3"}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for addString attribute, %s should be []string{\"1\", \"2\", \"3\", \"3\"}", s)
			}
		}

		{
			itf, ok := object["removeInt"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute removeInt missing")
			}

			sitf := itf.([]interface{})
			s := make([]int, len(sitf))
			for i := range sitf {
				s[i] = int(sitf[i].(float64))
			}

			if !intSlicesAreEqual(s, []int{1, 2}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for removeInt attribute, %v should be []int{1, 2}", s)
			}
		}

		{
			itf, ok := object["removeString"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute removeString missing")
			}

			sitf := itf.([]interface{})
			s := make([]string, len(sitf))
			for i := range sitf {
				s[i] = sitf[i].(string)
			}

			if !stringSlicesAreEqual(s, []string{"1", "2"}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for removeString attribute, %s should be []string{\"1\", \"2\"}", s)
			}
		}

		{
			itf, ok := object["addUniqueInt"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute addUniqueInt missing")
			}

			sitf := itf.([]interface{})
			s := make([]int, len(sitf))
			for i := range sitf {
				s[i] = int(sitf[i].(float64))
			}

			if !intSlicesAreEqual(s, []int{1, 2, 3}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for addUniqueInt attribute, %v should be []int{1, 2, 3}", s)
			}
		}

		{
			itf, ok := object["addUniqueString"]
			if !ok {
				t.Fatalf("TestBatchPartialUpdate: Attribute addUniqueString missing")
			}

			sitf := itf.([]interface{})
			s := make([]string, len(sitf))
			for i := range sitf {
				s[i] = sitf[i].(string)
			}

			if !stringSlicesAreEqual(s, []string{"1", "2", "3"}) {
				t.Errorf("TestBatchPartialUpdate: Wrong slice for addUniqueString attribute, %s should be []string{\"1\", \"2\", \"3\"}", s)
			}
		}
	}
}

func TestBatchMissingObjectIDs(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestBatchMissingObjectIDs")

	for _, action := range []string{
		"updateObject",
		"partialUpdateObject",
		"partialUpdateObjectNoCreate",
		"deleteObject",
	} {
		_, err := i.Batch([]BatchOperation{
			BatchOperation{Action: action, Body: map[string]interface{}{"key": "value"}},
		})
		if err == nil {
			t.Fatalf("Should return an error for a `%s` batch with a missing `objectID`", action)
		}
	}
}

func TestQueryRules(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestQueryRules")

	addRules(t, i, "TestQueryRules")

	t.Log("TestQueryRules: Retrieve all the added rules with multiple calls to GetRule")

	for _, ruleId := range []string{
		"brand_tagging",
		"remove_js",
		"substitute_coffee_with_tea",
	} {
		r, err := i.GetRule(ruleId)
		require.Nil(t, err, "should get rule without error")
		require.True(t, r.Enabled)
	}

	t.Log("TestQueryRules: Delete one query rule and check that it is not accessible anymore")
	{
		res, err := i.DeleteRule("brand_tagging", true)
		require.Nil(t, err, "should remove query rule without error")

		waitTask(t, i, res.TaskID)

		_, err = i.GetRule("brand_tagging")
		require.NotNil(t, err, "should not be able to get deleted rule")
	}

	t.Log("TestQueryRules: Search for a query rule with SearchRules")
	{
		params := Map{
			"query": "tea",
		}
		res, err := i.SearchRules(params)
		require.Nil(t, err, "should search for rules without error")
		require.Len(t, res.Hits, 1, "should only find one rule")
	}

	t.Log("TestQueryRules: Remove all existing rules with ClearRules and check that they are not accessible anymore")
	{
		res, err := i.ClearRules(true)
		require.Nil(t, err, "should clear all query rules without error")

		waitTask(t, i, res.TaskID)

		for _, ruleId := range []string{
			"remove_js",
			"substitute_coffee_with_tea",
		} {
			_, err = i.GetRule(ruleId)
			require.NotNil(t, err, "should not be able to get deleted rule")
		}
	}
}

func TestQueryRulesV2(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestQueryRulesV2")

	var tasks []int

	timeranges := []TimeRange{
		{
			From:  time.Date(2018, time.July, 24, 13, 35, 0, 0, time.UTC),
			Until: time.Date(2018, time.July, 25, 13, 35, 0, 0, time.UTC),
		},
		{
			From:  time.Date(2018, time.July, 26, 13, 35, 0, 0, time.UTC),
			Until: time.Date(2018, time.July, 27, 13, 35, 0, 0, time.UTC),
		},
	}

	expectedAutomaticFacetFilter := AutomaticFacetFilter{
		Facet:       "brand",
		Disjunctive: true,
		Score:       42,
	}

	t.Log("TestQueryRulesV2: Add disabled query rule with validity range")
	{
		rule := Rule{
			ObjectID:  "disabled_with_validity_range",
			Condition: NewSimpleRuleCondition(Contains, "{facet:brand}"),
			Consequence: RuleConsequence{
				Params: Map{
					"automaticFacetFilters": []AutomaticFacetFilter{expectedAutomaticFacetFilter},
				},
			},
			Validity:    timeranges,
			Description: "Automatic tagging of apple queries with apple brand",
		}
		rule.Disable()
		res, err := i.SaveRule(rule, false)
		require.NoError(t, err)
		tasks = append(tasks, res.TaskID)
	}

	expectedEdits := []Edit{
		DeleteEdit("android"),
	}

	t.Log("TestQueryRulesV2: Add disabled query rule with edits")
	{
		rule := Rule{
			ObjectID:  "disabled_with_delete_edit",
			Condition: NewSimpleRuleCondition(Contains, "android"),
			Consequence: RuleConsequence{
				Params: Map{
					"query": Map{
						"edits": expectedEdits,
					},
				},
			},
			Description: "Remove android from queries",
		}
		rule.Disable()
		res, err := i.BatchRules([]Rule{rule}, false, false)
		require.NoError(t, err)
		tasks = append(tasks, res.TaskID)
	}

	expectedHiddenObjects := []HiddenObject{
		{ObjectID: "42"},
		{ObjectID: "43"},
	}

	t.Log("TestQueryRulesV2: Add query rule with HiddenObject consequence")
	{
		rule := Rule{
			ObjectID:  "with_hidden_object",
			Condition: NewSimpleRuleCondition(Contains, "iphone"),
			Consequence: RuleConsequence{
				Hide: expectedHiddenObjects,
			},
			Description: "Remove objects 42 and 43 for iphone queries",
		}
		res, err := i.BatchRules([]Rule{rule}, false, false)
		require.NoError(t, err)
		tasks = append(tasks, res.TaskID)
	}

	waitTasksAsync(t, i, tasks)

	t.Log("TestQueryRulesV2: Check disabled query rule with validity range")
	{
		rule, err := i.GetRule("disabled_with_validity_range")
		require.NoError(t, err)
		require.False(t, rule.Enabled)
		require.Len(t, rule.Validity, len(timeranges))

		for _, expectedTimeRange := range timeranges {
			found := false
			for _, timeRange := range rule.Validity {
				if expectedTimeRange.From.Equal(timeRange.From) &&
					expectedTimeRange.Until.Equal(timeRange.Until) {
					found = true
				}
			}
			if !found {
				t.Errorf("missing TimeRange %#v from returned query rule %#v\n", expectedTimeRange, rule)
			}
		}

		automaticFacetFiltersItf, ok := rule.Consequence.Params["automaticFacetFilters"]
		require.True(t, ok)

		automaticFacetFilters, ok := automaticFacetFiltersItf.([]interface{})
		require.True(t, ok)

		require.Len(t, automaticFacetFilters, 1)

		var automaticFacetFilter AutomaticFacetFilter
		err = mapstructure.Decode(automaticFacetFilters[0], &automaticFacetFilter)
		require.NoError(t, err)
		require.Equal(t, automaticFacetFilter, expectedAutomaticFacetFilter)
	}

	t.Log("TestQueryRulesV2: Check disabled query rule with edits")
	{
		rule, err := i.GetRule("disabled_with_delete_edit")
		require.NoError(t, err)
		require.False(t, rule.Enabled)

		queryItf, ok := rule.Consequence.Params["query"]
		require.True(t, ok)

		var queryMap map[string][]Edit
		err = mapstructure.Decode(queryItf, &queryMap)
		require.NoError(t, err)

		require.Equal(t, expectedEdits, queryMap["edits"])

	}

	t.Log("TestQueryRulesV2: Check query rule with HiddenObject consequence")
	{
		rule, err := i.GetRule("with_hidden_object")
		require.NoError(t, err)
		require.True(t, rule.Enabled)
		require.Equal(t, expectedHiddenObjects, rule.Consequence.Hide)
	}
}

func TestBrowseAll(t *testing.T) {
	t.Parallel()
	_, i := initClientAndIndex(t, "TestBrowseAll")

	var tasks []int

	t.Log("TestBrowseAll: Add 3500 records")
	{
		objects := make([]Object, 500)
		for j := range objects {
			objects[j] = Object{"key": "value"}
		}

		for j := 0; j < 7; j++ {
			res, err := i.AddObjects(objects)
			require.Nil(t, err, "should add objects without error")
			tasks = append(tasks, res.TaskID)
		}
	}

	waitTasksAsync(t, i, tasks)

	t.Log("TestBrowseAll: Retrieve all of the 3500 records")
	{
		it, err := i.BrowseAll(nil)
		require.Nil(t, err, "should instantiate a new iterator without error")

		var count int

		for err == nil {
			_, err = it.Next()
			count++
		}

		require.Equal(t, NoMoreHitsErr.Error(), err.Error(), "should only return an \"end of result\"-kind error")
		require.Equal(t, 3501, count, "should browse all the records")
	}
}
