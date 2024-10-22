// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import "gitlab.waterfall.network/waterfall/protocol/gwat/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	"enode://89c1056d25113169b08b2134eb01782f918db49779c3cbd89b9115ced96533ad88ec47030ab4febc80a0fc7761a2de67fc8bf5d83d31a2c6f51f1d891ab0785a@34.38.132.50:30301",
	"enode://86495dd3160c2dbec138079fe99e66c1fdcbd458de3468f19bc123e5090aefea5d16a6d498eb617fb34b8d9e548369320dc315d38c9ffb84aedcec5429b8cbda@165.22.22.103:30301",
	"enode://46e414d0a254502c57df22a621d334229082aaf356395522070662c27c281770607ba72e71c45feb8e0114a585a0c35463a463efffea7e1d191e515d60677b21@68.183.198.62:30301",
	"enode://167ed73438a38d31c84aea4823b4c518c80ab28d36129470aa193b33d9501c93ed0e7cbea3d2f2944ea49f3ba18768a0b71642b5f2294063dd40e2242ba10d60@143.110.184.101:30301",
	"enode://fd17aa5114181b97d24a64dbb99da21aa6c004c56633a0870c932656d62092c9eec4ebff4f52ad49b3ac5c08edf5577e2b76502d7a29c8746bdb882291563d4e@78.47.100.242:30301",
	"enode://bd7d11d74c9283fc3f8eeb76ea94ee6cd5ee1131012b6974dafe7eabbe45e7776ec3630ceb9cd8a8b9a935537c64f21ae446f2651018ed6e87b8f561a7ef89d6@65.108.217.99:30301",
	"enode://164309032a3b15c960d5071cee2666ed81f78f21162efeb5c1b53df78f7ce336574d65b93f41a31785f72b024d342d13da8c14fe2fc10edf44b938e172ffd23d@34.38.233.229:30301",
	"enode://cda076786525d4b533c651d56a812ebeffde561b63ad9878f0a7869ab17334d057eb483f6cb18ef20f2cb598823886916e7fe511e8f770be3807d19cc26a20c6@34.47.4.2:30301",
	"enode://8665f9d35d3c5e6b220b16a27e8cfd50624568278847f31913d00586be562c777742ebdba73bfed5ba24ca990533402b57badcc683119f14da444bcc98ed798f@34.151.198.106:30301",
	"enode://d6d19ea1f4f2182f4119b2968a51b8c90f5a01f3ed5032e852dc2a92d73c1fe90c991ff0961806a089c78b2d5e9fd4284f79bd4b03cc0a6808d35c34755c2967@34.18.19.75:30301",
	"enode://19b05cae6d2fb05eb0729af450a5aab4912c2cd43690070f81b6eab956f3458deeaa1e13a4cfbd4ce831448fbd3b7b0fad8f12f620b93681230493537cc97158@34.165.48.114:30301",
	"enode://73100a8d3fca14a2403bdc9fcfa2a9a2f9019b5b4562157ed0e97fa924751859a3914ad7b76c9aaef427512698f291feda1ad7b88d9e4c82b14b86f59a98fb0e@34.35.57.44:30301",
}

// Testnet8Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// testnet8 test network.
var Testnet8Bootnodes = []string{
	"enode://716898aedc2337bc1f8c2a936f4b1080e5c4794ba55b31d4cf5898d02dd036150debb742f9929f6f6a7030afaf324604fda57702769ad05bdcce526b3b12cbf1@128.140.45.145:30301",
	"enode://367797cdca79faffab404c6f3f8137511aa4afb7cbd47d51a0be6966f931bd9204346e44016ba1e2435a8067333de4d98e2e5a4b09cf1fabc75f6909e6f0db30@34.47.9.74:30301",
	"enode://fb08dcc8e81c89aeccfda05bf95351d83c91847c7741ee60f1fcc54cc15b63720eb52b1dbb62e48f386388c6786b9c5fb78e324a6404fdf157f0f19955907146@34.130.246.209:30301",
	"enode://64f5b1f18caf5665ba5762c7c298a820aa8eec024ce0c9093a8727ada5b97631b8b2b1bb3fbb811ead063fd8d4920001f7c23b27f82ca527e8a88fe696b411ed@35.198.58.167:30301",
	"enode://89e726ff8d68bac89a382e9f037a49da0364c231a2c7b5ffd9604cbf514e35d8c7a81224ea0b487c0ce6fce86052c7e89e7d9d346af3dbfc86f6f78dc269af63@34.176.93.86:30301",
	"enode://19102f79a69ff0e65f2bb0d0e7093dd118c7f3c9a8eec073bfdf6bafadcc02b07f2f6fbfa88b53bed9650c0c703d99303f7d221c77a41438faf9f005938793f5@34.18.82.149:30301",
	"enode://c815726f13a941e1f744183d3bdc1114097e810bd91acaf2a8345ab052dc45703c815047d92971cfc6852e358e7e57e5306138a71f117b6dfec332c674588df2@34.166.40.43:30301",
	"enode://79b1d6ce17e55e59dfec4bea7675e21c25bbcaec42e66fd70bbc57675bd44997c9732dafba24060eea7189ae73272ccae2834509b8040835836036c3e66428d1@34.165.31.150:30301",
	"enode://d9c2702e351f4066e0699966880706734b97e9ad307d51530d2f9f7f83daaa5a26472d0880257796cb76a4a1cc5df2615846840a12a887bff8780de9a76dcfa7@34.35.38.213:30301",
	"enode://18a250701d0c6d73bf1f8d285172412d6e6aa455f138dce585e57dce0cad18fb4cd0470838b226fc027fc2d7df384374965ba3efa80357aee36c9bcd78fefa36@34.142.104.218:30301",
	"enode://93546973a7d5240d2d9a46f91811461bdf1caadf9aff197b3f08a82a4762e52b7ffda0eac44c5499014aaa97ec52a63cd4e671495219728197660d0759e11794@34.175.54.201:30301",
}

var V5Bootnodes = []string{
	// Teku team's bootnode
	"enr:-KG4QOtcP9X1FbIMOe17QNMKqDxCpm14jcX5tiOE4_TyMrFqbmhPZHK_ZPG2Gxb1GE2xdtodOfx9-cgvNtxnRyHEmC0ghGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQDE8KdiXNlY3AyNTZrMaEDhpehBDbZjM_L9ek699Y7vhUJ-eAdMyQW_Fil522Y0fODdGNwgiMog3VkcIIjKA",
	"enr:-KG4QDyytgmE4f7AnvW-ZaUOIi9i79qX4JwjRAiXBZCU65wOfBu-3Nb5I7b_Rmg3KCOcZM_C3y5pg7EBU5XGrcLTduQEhGV0aDKQ9aX9QgAAAAD__________4JpZIJ2NIJpcIQ2_DUbiXNlY3AyNTZrMaEDKnz_-ps3UUOfHWVYaskI5kWYO_vtYMGYCQRAR3gHDouDdGNwgiMog3VkcIIjKA",
	// Prylab team's bootnodes
	"enr:-Ku4QImhMc1z8yCiNJ1TyUxdcfNucje3BGwEHzodEZUan8PherEo4sF7pPHPSIB1NNuSg5fZy7qFsjmUKs2ea1Whi0EBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQOVphkDqal4QzPMksc5wnpuC3gvSC8AfbFOnZY_On34wIN1ZHCCIyg",
	"enr:-Ku4QP2xDnEtUXIjzJ_DhlCRN9SN99RYQPJL92TMlSv7U5C1YnYLjwOQHgZIUXw6c-BvRg2Yc2QsZxxoS_pPRVe0yK8Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMeFF5GrS7UZpAH2Ly84aLK-TyvH-dRo0JM1i8yygH50YN1ZHCCJxA",
	"enr:-Ku4QPp9z1W4tAO8Ber_NQierYaOStqhDqQdOPY3bB3jDgkjcbk6YrEnVYIiCBbTxuar3CzS528d2iE7TdJsrL-dEKoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpD1pf1CAAAAAP__________gmlkgnY0gmlwhBLf22SJc2VjcDI1NmsxoQMw5fqqkw2hHC4F5HZZDPsNmPdB1Gi8JPQK7pRc9XHh-oN1ZHCCKvg",
	// Lighthouse team's bootnodes
	"enr:-IS4QLkKqDMy_ExrpOEWa59NiClemOnor-krjp4qoeZwIw2QduPC-q7Kz4u1IOWf3DDbdxqQIgC4fejavBOuUPy-HE4BgmlkgnY0gmlwhCLzAHqJc2VjcDI1NmsxoQLQSJfEAHZApkm5edTCZ_4qps_1k_ub2CxHFxi-gr2JMIN1ZHCCIyg",
	"enr:-IS4QDAyibHCzYZmIYZCjXwU9BqpotWmv2BsFlIq1V31BwDDMJPFEbox1ijT5c2Ou3kvieOKejxuaCqIcjxBjJ_3j_cBgmlkgnY0gmlwhAMaHiCJc2VjcDI1NmsxoQJIdpj_foZ02MXz4It8xKD7yUHTBx7lVFn3oeRP21KRV4N1ZHCCIyg",
	// EF bootnodes
	"enr:-Ku4QHqVeJ8PPICcWk1vSn_XcSkjOkNiTg6Fmii5j6vUQgvzMc9L1goFnLKgXqBJspJjIsB91LTOleFmyWWrFVATGngBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhAMRHkWJc2VjcDI1NmsxoQKLVXFOhp2uX6jeT0DvvDpPcU8FWMjQdR4wMuORMhpX24N1ZHCCIyg",
	"enr:-Ku4QG-2_Md3sZIAUebGYT6g0SMskIml77l6yR-M_JXc-UdNHCmHQeOiMLbylPejyJsdAPsTHJyjJB2sYGDLe0dn8uYBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhBLY-NyJc2VjcDI1NmsxoQORcM6e19T1T9gi7jxEZjk_sjVLGFscUNqAY9obgZaxbIN1ZHCCIyg",
	"enr:-Ku4QPn5eVhcoF1opaFEvg1b6JNFD2rqVkHQ8HApOKK61OIcIXD127bKWgAtbwI7pnxx6cDyk_nI88TrZKQaGMZj0q0Bh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDayLMaJc2VjcDI1NmsxoQK2sBOLGcUb4AwuYzFuAVCaNHA-dy24UuEKkeFNgCVCsIN1ZHCCIyg",
	"enr:-Ku4QEWzdnVtXc2Q0ZVigfCGggOVB2Vc1ZCPEc6j21NIFLODSJbvNaef1g4PxhPwl_3kax86YPheFUSLXPRs98vvYsoBh2F0dG5ldHOIAAAAAAAAAACEZXRoMpC1MD8qAAAAAP__________gmlkgnY0gmlwhDZBrP2Jc2VjcDI1NmsxoQM6jr8Rb1ktLEsVcKAPa08wCsKUmvoQ8khiOl_SLozf9IN1ZHCCIyg",
}

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	return ""
	//// todo fix required
	//var net string
	//switch genesis {
	//case MainnetGenesisHash:
	//	net = "mainnet"
	//case Testnet8GenesisHash:
	//	net = "testnet8"
	//default:
	//	return ""
	//}
	//return "enrtree://AKA3AM6LPBYEUDMVNU3BSVQJ5AD45Y7YPOHJLEF6W26QOE4VTUDPE@" + protocol + "." + net + ".ethdisco.net"
}
