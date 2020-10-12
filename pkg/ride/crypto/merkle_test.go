package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerkleRootHash(t *testing.T) {
	root, err := base64.StdEncoding.DecodeString("eh9fm3HeHZ3XA/UfMpC9HSwLVMyBLgkAJL0MIVBIoYk=")
	require.NoError(t, err)
	for _, test := range []struct {
		leaf  string
		proof string
	}{
		{"AAAdIQ==", "ACCP8jyg8Rv62mE4IMD4FGATnUXEIoCIK0LMoQCjAGpl5AEg16lhBiAz+xB8hwUs8U7dTJeGmJQyWVfXmHqzA+b2YuUBICJEors9RDiMZNeWp2yIlJrpf/a4rZxTvI7yIx3D5pihACAaVrwYIveDbOb3uE+Hj1w+Tl0vornHqPT9pCja/TmfPgAgxGoHWeIYY3RDkfAyYD99LA6OXdiXaB9a86EifTMS728AINbkCaDKCXEc5i61+c3ewBPFoCCYMCyvIrDbmHAThKt4ACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw=="},
		{"AAAc6w==", "ACDdSC04SpOqrUb7PbWs5NaLSSm/k6d1eG0MgFwTDEeJXAAg0iC2Dfqsu4tJUQt+xiDjvHyxUVu664rKruVL8zs6c60AIKLhp/AFQkokTe/NMQnKFL5eTMvDlFejApmJxPY6Rp8XACAWrdgB8DwvPA8D04E9HgUjhKghAn5aqtZnuKcmpLHztQAgd2OG15WYz90r1WipgXwjdq9WhvMIAtvGlm6E3WYY12oAIJXPPVIdbwOTdUJvCgMI4iape2gvR55vsrO2OmJJtZUNASAya23YyBl+EpKytL9+7cPdkeMMWSjk0Bc0GNnqIisofQ=="},
		{"AAAc+w==", "ASADLSXbJGHQ7MMNaAqIfuLAwkvd7pQNnSQKcRnd3TYA0gAgNqksHYDS1xq5mKOpcWhxdM9KtzAJwVlJ8RECYsm9PMkAIEYOaapf0SZM4wZS8nZ95byib0SgjBLy1XG676X6lvoAASBOVhj3XzjWhqziBwKr/2M6v9VYF026vuWwXieZWMUdSwEgPqfL+ywsEjtOpywTh+k4zz23LGD2KGWHqfJvD8/9WdgBICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw=="},
		{"AAAIVw==", "ACBlQ+wlERW7AiK0dPotu7wLCCaMcH+X2D9XEU+D8TSNbwEgld8vUreEqWpiFo0nMwUsiP6LPhi8XWpV6Gge/3edo5MBIFCGuyg86lVn9ga7hNacZPBNd6T5gtMk+5OWpO8HthAmASDPIhoSPwQ9YL5aa+S6MjaLNe74dY3/Mq/OrpP7C46/8wAg1FSDEXwBdMgQkmK245kByRV39HfsgpmTdbbYd85GqI0BICdQYY0pkNys0gKNdIzTMj3Ou1Ags2EgP237fvxZqR9yACAUkUex5ycLaviKxbHHkaC563PXFUWouAlN7c1xjz98Sw=="},
	} {
		leaf, err := base64.StdEncoding.DecodeString(test.leaf)
		require.NoError(t, err)
		proof, err := base64.StdEncoding.DecodeString(test.proof)
		require.NoError(t, err)
		actual, err := MerkleRootHash(leaf, proof)
		require.NoError(t, err)
		assert.ElementsMatch(t, root, actual)
	}
}
