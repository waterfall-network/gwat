// Copyright 2024   Blue Wave Inc.
//
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

//go:build dummy
// +build dummy

// This file is part of a workaround for `go mod vendor` which won't vendor
// C files if there's no Go file in the same directory.
// This would prevent the crypto/secp256k1/libsecp256k1/include/secp256k1.h file to be vendored.
//
// This Go file imports the c directory where there is another dummy.go file which
// is the second part of this workaround.
//
// These two files combined make it so `go mod vendor` behaves correctly.
//
// See this issue for reference: https://github.com/golang/go/issues/26366

package secp256k1

import (
	_ "gitlab.waterfall.network/waterfall/protocol/gwat/crypto/secp256k1/libsecp256k1/include"
	_ "gitlab.waterfall.network/waterfall/protocol/gwat/crypto/secp256k1/libsecp256k1/src"
	_ "gitlab.waterfall.network/waterfall/protocol/gwat/crypto/secp256k1/libsecp256k1/src/modules/recovery"
)
