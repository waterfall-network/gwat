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

//go:build ((linux && amd64) || (linux && arm64) || (darwin && amd64) || (darwin && arm64) || (windows && amd64)) && !blst_disabled
// +build linux,amd64 linux,arm64 darwin,amd64 darwin,arm64 windows,amd64
// +build !blst_disabled

package bls_sig

import blst "github.com/supranational/blst/bindings/go"

// Internal types for blst.
type blstPublicKey = blst.P1Affine
type blstSignature = blst.P2Affine

//type blstAggregateSignature = blst.P2Aggregate
//type blstAggregatePublicKey = blst.P1Aggregate
