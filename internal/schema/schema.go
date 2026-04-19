// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: GPL-3.0-or-later

package schema

import _ "embed"

//go:embed schema.sql
var Schema string

//go:embed seed_edge_types.sql
var EdgeTypeSeed string
