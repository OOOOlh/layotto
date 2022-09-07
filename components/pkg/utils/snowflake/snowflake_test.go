//
// Copyright 2021 Layotto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package snowflake

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSnowflakeRingBufferMetadataBitsSum(t *testing.T) {
	properties := make(map[string]string)
	properties["workerBits"] = "1"
	properties["timeBits"] = "1"
	properties["seqBits"] = "1"

	_, err := ParseSnowflakeRingBufferMetadata(properties)
	assert.Error(t, err)
}

func TestParseSnowflakeRingBufferMetadataTimeFormat(t *testing.T) {
	properties := make(map[string]string)
	properties["startTime"] = "2022.01.01"

	_, err := ParseSnowflakeRingBufferMetadata(properties)
	assert.Error(t, err)
}

func TestParseSnowflakeMysqlMetadata(t *testing.T) {
	properties := make(map[string]string)
	properties["tableName"] = "layotto_sequence_snowflake"

	_, err := ParseSnowflakeMysqlMetadata(properties)
	assert.Error(t, err)
}
