package ride

import (
	"github.com/stretchr/testify/require"

	"testing"
)

func TestDecompiler(t *testing.T) {
	t.Run("check match", func(t *testing.T) {
		source := `AAIDAAAAAAAAAAIIAQAAAAEBAAAADmdldE51bWJlckJ5S2V5AAAAAQAAAANrZXkEAAAAByRtYXRjaDAJAAQaAAAAAgUAAAAEdGhpcwUAAAADa2V5AwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAAANJbnQEAAAAAWEFAAAAByRtYXRjaDAFAAAAAWEAAAAAAAAAAAAAAAAAAAAAAFURfiw=`
		tree, err := ParseB64(source)
		require.NoError(t, err)

		require.Equal(t,
			"func getNumberByKey(key) { match getInteger(this,key) { case a: Int => { a } case _ => { 0 } }",
			DecompileTree(tree),
		)
	})
	t.Run("check match multiple choices", func(t *testing.T) {
		/*
			{-# STDLIB_VERSION 3 #-}
			{-# SCRIPT_TYPE ACCOUNT #-}
			{-# CONTENT_TYPE DAPP #-}
			func getNumberByKey (key: Transaction) = match key {
			    case a: BurnTransaction => 1
			    case a: IssueTransaction => 2
			    case _ => 0
			}
		*/
		source := `AAIDAAAAAAAAAAIIAQAAAAEBAAAADmdldE51bWJlckJ5S2V5AAAAAQAAAANrZXkEAAAAByRtYXRjaDAFAAAAA2tleQMJAAABAAAAAgUAAAAHJG1hdGNoMAIAAAAPQnVyblRyYW5zYWN0aW9uBAAAAAFhBQAAAAckbWF0Y2gwAAAAAAAAAAABAwkAAAEAAAACBQAAAAckbWF0Y2gwAgAAABBJc3N1ZVRyYW5zYWN0aW9uBAAAAAFhBQAAAAckbWF0Y2gwAAAAAAAAAAACAAAAAAAAAAAAAAAAAAAAAADkJzDk`
		tree, err := ParseB64(source)
		require.NoError(t, err)

		require.Equal(t,
			"func getNumberByKey(key) { match key { case a: BurnTransaction => { 1 } case a: IssueTransaction => { 2 } case _ => { 0 } }",
			DecompileTree(tree),
		)
	})
}
