/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package names

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleNameGenerator(t *testing.T) {
	t.Run("GeneratesNameWithPrefix", func(t *testing.T) {
		name := SimpleNameGenerator.GenerateName("foo")

		assert.True(t, strings.HasPrefix(name, "foo"))
		assert.NotEqual(t, "foo", name)
	})

	t.Run("GeneratesNameWithMaxLength", func(t *testing.T) {
		base := strings.Repeat("a", MaxGeneratedNameLength)

		name := SimpleNameGenerator.GenerateName(base)

		assert.True(t, strings.HasPrefix(name, base))
		assert.NotEqual(t, base, name)
		assert.LessOrEqual(t, len(name), maxNameLength)
	})

	t.Run("TrimsBaseNameIfTooLong", func(t *testing.T) {
		base := strings.Repeat("a", maxNameLength)

		name := SimpleNameGenerator.GenerateName(base)

		assert.True(t, strings.HasPrefix(name, base[:MaxGeneratedNameLength]))
		assert.NotEqual(t, base[:MaxGeneratedNameLength], name)
		assert.LessOrEqual(t, len(name), maxNameLength)
	})
}

func TestString(t *testing.T) {
	t.Run("GeneratesStringOfCorrectLength", func(t *testing.T) {
		length := 10

		str := String(length)

		assert.Equal(t, length, len(str))
	})

	t.Run("GeneratesAlphanumericStringWithoutVowels", func(t *testing.T) {
		length := 10

		str := String(length)

		for _, char := range str {
			assert.Contains(t, alphanums, string(char))
		}
	})

	t.Run("HandlesRemainingZeroCondition", func(t *testing.T) {
		length := maxAlphanumsPerInt + 1 // Ensure we hit the remaining == 0 condition

		str := String(length)

		assert.Equal(t, length, len(str))
		for _, char := range str {
			assert.Contains(t, alphanums, string(char))
		}
	})
}
