// Copyright © 2017 Joyent, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pg_test

import (
	"testing"

	"github.com/bschofield/pg_prefaulter/pg"
	"github.com/kylelemons/godebug/pretty"
)

// Precompute the expected results from constants
func TestWAL_Constants(t *testing.T) {
	if diff := pretty.Compare(pg.WALPageSize, 8192); diff != "" {
		t.Fatalf("WALPageSize diff: (-got +want)\n%s", diff)
	}

	if diff := pretty.Compare(pg.WALSegmentSize, 16777216); diff != "" {
		t.Fatalf("WALSegmentSize diff: (-got +want)\n%s", diff)
	}

	if diff := pretty.Compare(pg.WALSegmentsPerWALID, 256); diff != "" {
		t.Fatalf("WALSegmentsPerWALID diff: (-got +want)\n%s", diff)
	}
}
