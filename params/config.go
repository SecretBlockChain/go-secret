// Copyright 2016 The go-ethereum Authors
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

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/SecretBlockChain/go-secret/common"
	"github.com/SecretBlockChain/go-secret/common/math"
	"github.com/SecretBlockChain/go-secret/crypto"
)

//go:generate gencodec -type EqualityReward -field-override equalityRewardMarshaling -out gen_equality_reward.go
//go:generate gencodec -type EqualityConfig -field-override equalityConfigMarshaling -out gen_equality_config.go

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash = common.HexToHash("0x9759272a3cdd7583edc93ae51475b18efb7bbc3d1934189d87ac4a6d2493e6b5")
	TestnetGenesisHash = common.HexToHash("0xf024c8cb7b92ea396317b9a64afec723c48fc8df1843ce02c171daf95a5b7dc0")
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of
// the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]*TrustedCheckpoint{
	//MainnetGenesisHash: MainnetTrustedCheckpoint,
}

// CheckpointOracles associates each known checkpoint oracles with the genesis hash of
// the chain it belongs to.
var CheckpointOracles = map[common.Hash]*CheckpointOracleConfig{
	//MainnetGenesisHash: MainnetCheckpointOracle,
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(211),
		HomesteadBlock:      big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		Equality:            MainNetEqualityConfig(),
	}

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	//MainnetTrustedCheckpoint = &TrustedCheckpoint{
	//	SectionIndex: 336,
	//	SectionHead:  common.HexToHash("0xd42b78902b6527a80337bf1bc372a3ccc3db97e9cc7cf421ca047ae9076c716b"),
	//	CHTRoot:      common.HexToHash("0xd97f3b30f7e0cb958e4c67c53ec27745e5a165e33e56821b86523dfee62b783a"),
	//	BloomRoot:    common.HexToHash("0xf3cbfd070fababfe2adc9b23fc02c731f6ca2cce6646b3ede4ef2db06092ccce"),
	//}

	// MainnetCheckpointOracle contains a set of configs for the main network oracle.
	//MainnetCheckpointOracle = &CheckpointOracleConfig{
	//	Address: common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"),
	//	Signers: []common.Address{
	//		common.HexToAddress("0x1b2C260efc720BE89101890E4Db589b44E950527"), // Peter
	//		common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
	//		common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
	//		common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
	//		common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
	//	},
	//	Threshold: 2,
	//}

	// TestnetChainConfig contains the chain parameters to run a node on the test network.
	TestnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(21610),
		HomesteadBlock:      big.NewInt(0),
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000"),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		Equality:            TestnetEqualityConfig(),
	}

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, nil, new(EthashConfig), nil, nil}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, nil, nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil}

	TestChainConfig = &ChainConfig{big.NewInt(1), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, nil, nil, new(EthashConfig), nil, nil}
	TestRules       = TestChainConfig.Rules(new(big.Int))
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// HashEqual returns an indicator comparing the itself hash with given one.
func (c *TrustedCheckpoint) HashEqual(hash common.Hash) bool {
	if c.Empty() {
		return hash == common.Hash{}
	}
	return c.Hash() == hash
}

// Hash returns the hash of checkpoint's four key fields(index, sectionHead, chtRoot and bloomTrieRoot).
func (c *TrustedCheckpoint) Hash() common.Hash {
	buf := make([]byte, 8+3*common.HashLength)
	binary.BigEndian.PutUint64(buf, c.SectionIndex)
	copy(buf[8:], c.SectionHead.Bytes())
	copy(buf[8+common.HashLength:], c.CHTRoot.Bytes())
	copy(buf[8+2*common.HashLength:], c.BloomRoot.Bytes())
	return crypto.Keccak256Hash(buf)
}

// Empty returns an indicator whether the checkpoint is regarded as empty.
func (c *TrustedCheckpoint) Empty() bool {
	return c.SectionHead == (common.Hash{}) || c.CHTRoot == (common.Hash{}) || c.BloomRoot == (common.Hash{})
}

// CheckpointOracleConfig represents a set of checkpoint contract(which acts as an oracle)
// config which used for light client checkpoint syncing.
type CheckpointOracleConfig struct {
	Address   common.Address   `json:"address"`
	Signers   []common.Address `json:"signers"`
	Threshold uint64           `json:"threshold"`
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)

	YoloV1Block *big.Int `json:"yoloV1Block,omitempty"` // YOLO v1: https://github.com/ethereum/EIPs/pull/2657 (Ephemeral testnet)
	EWASMBlock  *big.Int `json:"ewasmBlock,omitempty"`  // EWASM switch block (nil = no fork, 0 = already activated)

	// Various consensus engines
	Ethash   *EthashConfig   `json:"ethash,omitempty"`
	Clique   *CliqueConfig   `json:"clique,omitempty"`
	Equality *EqualityConfig `json:"equality,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// EqualityReward is reward rule of mint block.
type EqualityReward struct {
	Number uint64   `json:"number"`                     // Block number
	Reward *big.Int `json:"reward" gencodec:"required"` // Token reward of mint block
}

type EqualityRewards []EqualityReward

// EqualityConfig is the consensus engine configs for proof-of-equality based sealing.
type EqualityConfig struct {
	Period              uint64           `json:"period"`                                  // Number of seconds between blocks to enforce
	Epoch               uint64           `json:"epoch"`                                   // Epoch length to reset votes and checkpoint
	MaxValidatorsCount  uint64           `json:"maxValidatorsCount"`                      // Max count of validators
	MinCandidateBalance *big.Int         `json:"minCandidateBalance" gencodec:"required"` // Min candidate balance to valid this candidate
	GenesisTimestamp    uint64           `json:"genesisTimestamp"`                        // The timestamp of first Block
	Validators          []common.Address `json:"validators"`                              // Genesis validator list
	Pool                common.Address   `json:"pool"`                                    // Deposit pool address
	Rewards             EqualityRewards  `json:"rewards"`                                 // Reward rule of mint block
}

type equalityRewardMarshaling struct {
	Number uint64
	Reward *math.HexOrDecimal256
}

type equalityConfigMarshaling struct {
	Period              uint64
	Epoch               uint64
	MaxValidatorsCount  uint64
	MinCandidateBalance *math.HexOrDecimal256
	GenesisTimestamp    uint64
	Validators          []common.Address
	Pool                common.Address
	Rewards             EqualityRewards
}

// MainNetEqualityConfig returns mainnet config of equality consensus engine.
func MainNetEqualityConfig() *EqualityConfig {
	reward, _ := big.NewInt(0).SetString("1bc16d674ec80000", 16)
	minCandidateBalance, _ := big.NewInt(0).SetString("3635c9adc5dea00000", 16)

	return &EqualityConfig{
		Period:              3,
		Epoch:               28800,
		MaxValidatorsCount:  21,
		MinCandidateBalance: minCandidateBalance,
		GenesisTimestamp:    1625976000,
		Pool:                common.HexToAddress("0x53d77827bE168aB2a911B5A14D0f16D1C5657196"),
		Rewards: []EqualityReward{
			{
				Number: 45000000,
				Reward: reward,
			},
			{
				Number: 45000001,
				Reward: big.NewInt(0),
			},
		},
		Validators: []common.Address{
			common.HexToAddress("0xBBaC30738185396586C839232eDB9508Ff4Afe88"),
			common.HexToAddress("0x6756b7E36fa2CE9614879B4849286C54a46C9E3D"),
			common.HexToAddress("0x84cb756dB6c0fC1a36E6F2b76dF06916e6455f1C"),
			common.HexToAddress("0x8830Df43C3C63b33f26E341a604aEe5D049E6C2C"),
			common.HexToAddress("0x4917129800B4223FaE89E8B66a6a9F7400F3556b"),
			common.HexToAddress("0x9054C3998e4255c47dec34D14A7197F2302a8fE3"),
			common.HexToAddress("0xFe90133EE1dcda1F9B9Aeb79E8fd3717945179B2"),
			common.HexToAddress("0xAB82F5833C8E0C091e3D27E2B6906d909122366A"),
			common.HexToAddress("0xa5Af52D214591E4c5bb592038DcE546F73F3dC73"),
			common.HexToAddress("0x775e3fF7d0F9BD0956Ed12E911c50410Bd0df9cB"),
			common.HexToAddress("0xeB4efEE5b099edAbD8d3733986b1C3c064a4583e"),
			common.HexToAddress("0x955334d7Ab6B5Fb5Cb3ed62A26D623b97daaF09C"),
			common.HexToAddress("0xe18EB7ab2DB20fF93A54Db3a3806c0a95a863277"),
			common.HexToAddress("0xF70ECA281539DEF0ff7b8d38D0328e3e82F91f76"),
			common.HexToAddress("0x89A22a4066f247f058b0fB14a0449D350aD88382"),
			common.HexToAddress("0x5aB35cA3648Df46B8EF70eB35Ff5242e52F2938b"),
			common.HexToAddress("0x17f694c4786bD16A10E8B990A42ad233491cf033"),
			common.HexToAddress("0x03520937b4b2dB27a9Ba30c9c09d99aAE36f870e"),
			common.HexToAddress("0xCD5843479eb2056dDE3170E9611F1eEFBf33b90a"),
			common.HexToAddress("0x909c396d2635351456c093B87eE8eb61bB85D970"),
			common.HexToAddress("0xf141746840d77F4568aB60a6588D4e5F562A9c12"),
			common.HexToAddress("0x2D5d47ea275f36cD7E22dBabEdaD5D20e332734D"),
			common.HexToAddress("0xD0e694e5457cBa154211bB7701f4819Fe72b5391"),
			common.HexToAddress("0x2065B4A6a37D27237e39Ac6EF94D767a5EB879e5"),
			common.HexToAddress("0xad4318FdB74fa982d70c560385FE85270C515530"),
			common.HexToAddress("0x467298ceE63477056EBA376786195492C4E67247"),
			common.HexToAddress("0x7dC2dFf0676838B5fdD49222bd228564d47b68f3"),
			common.HexToAddress("0xb5BD5a4068138A452A736cC1afbdfE1F304E3090"),
			common.HexToAddress("0xc23f6A8681B9FEe1B545D906570C3eaB893eC22F"),
			common.HexToAddress("0xd41AC1C60cA3C65E1B85eb4Bc3a657A33092f570"),
			common.HexToAddress("0x0A9fEACd84DA88fE755a8E46B03b91666661aeeF"),
		},
	}
}

// TestnetEqualityConfig returns testnet config of equality consensus engine.
func TestnetEqualityConfig() *EqualityConfig {
	reward, _ := big.NewInt(0).SetString("1bc16d674ec80000", 16)
	minCandidateBalance, _ := big.NewInt(0).SetString("56bc75e2d63100000", 16)

	return &EqualityConfig{
		Period:              5,
		Epoch:               12,
		MaxValidatorsCount:  21,
		MinCandidateBalance: minCandidateBalance,
		GenesisTimestamp:    1623283200,
		Rewards: []EqualityReward{
			{
				Number: 45000000,
				Reward: reward,
			},
			{
				Number: 45000001,
				Reward: big.NewInt(0),
			},
		},
		Validators: []common.Address{
			common.HexToAddress("0x6c4ab069affd856bb915ee93cb59370574f5331e"),
			common.HexToAddress("0x6e935e0c8cf83aea41c807acfc00b8588cb56717"),
		},
	}
}

// String implements the stringer interface, returning the consensus engine details.
func (c *EqualityConfig) String() string {
	return "equality"
}

// Equal compares two EqualityConfigs for equal.
func (c *EqualityConfig) Equal(other EqualityConfig) bool {
	if c.Pool != other.Pool {
		return false
	}
	if c.Epoch != other.Epoch {
		return false
	}
	if c.Period != other.Period {
		return false
	}
	if c.MaxValidatorsCount != other.MaxValidatorsCount {
		return false
	}
	if c.MinCandidateBalance.Cmp(other.MinCandidateBalance) != 0 {
		return false
	}
	if c.GenesisTimestamp != other.GenesisTimestamp {
		return false
	}

	if len(c.Validators) != len(other.Validators) {
		return false
	}
	for idx, validator := range c.Validators {
		if validator != other.Validators[idx] {
			return false
		}
	}

	if len(c.Rewards) != len(other.Rewards) {
		return false
	}
	for idx, reward := range c.Rewards {
		if reward.Number != other.Rewards[idx].Number {
			return false
		}
		if reward.Reward.Cmp(other.Rewards[idx].Reward) != 0 {
			return false
		}
	}
	return true
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	case c.Equality != nil:
		engine = c.Equality
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Constantinople: %v Petersburg: %v Istanbul: %v, Muir Glacier: %v, YOLO v1: %v, Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.ConstantinopleBlock,
		c.PetersburgBlock,
		c.IstanbulBlock,
		c.MuirGlacierBlock,
		c.YoloV1Block,
		engine,
	)
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// IsMuirGlacier returns whether num is either equal to the Muir Glacier (EIP-2384) fork block or greater.
func (c *ChainConfig) IsMuirGlacier(num *big.Int) bool {
	return isForked(c.MuirGlacierBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return isForked(c.PetersburgBlock, num) || c.PetersburgBlock == nil && isForked(c.ConstantinopleBlock, num)
}

// IsIstanbul returns whether num is either equal to the Istanbul fork block or greater.
func (c *ChainConfig) IsIstanbul(num *big.Int) bool {
	return isForked(c.IstanbulBlock, num)
}

// IsYoloV1 returns whether num is either equal to the YoloV1 fork block or greater.
func (c *ChainConfig) IsYoloV1(num *big.Int) bool {
	return isForked(c.YoloV1Block, num)
}

// IsEWASM returns whether num represents a block number after the EWASM fork
func (c *ChainConfig) IsEWASM(num *big.Int) bool {
	return isForked(c.EWASMBlock, num)
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	type fork struct {
		name     string
		block    *big.Int
		optional bool // if true, the fork may be nil and next fork is still allowed
	}
	var lastFork fork
	for _, cur := range []fork{
		{name: "homesteadBlock", block: c.HomesteadBlock},
		{name: "daoForkBlock", block: c.DAOForkBlock, optional: true},
		{name: "eip150Block", block: c.EIP150Block},
		{name: "eip155Block", block: c.EIP155Block},
		{name: "eip158Block", block: c.EIP158Block},
		{name: "byzantiumBlock", block: c.ByzantiumBlock},
		{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		{name: "petersburgBlock", block: c.PetersburgBlock},
		{name: "istanbulBlock", block: c.IstanbulBlock},
		{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
		{name: "yoloV1Block", block: c.YoloV1Block},
	} {
		if lastFork.name != "" {
			// Next one must be higher number
			if lastFork.block == nil && cur.block != nil {
				return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %v",
					lastFork.name, cur.name, cur.block)
			}
			if lastFork.block != nil && cur.block != nil {
				if lastFork.block.Cmp(cur.block) > 0 {
					return fmt.Errorf("unsupported fork ordering: %v enabled at %v, but %v enabled at %v",
						lastFork.name, lastFork.block, cur.name, cur.block)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || cur.block != nil {
			lastFork = cur
		}
	}
	return nil
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainID, newcfg.ChainID) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, head) {
		// the only case where we allow Petersburg to be set in the past is if it is equal to Constantinople
		// mainly to satisfy fork ordering requirements which state that Petersburg fork be set if Constantinople fork is set
		if isForkIncompatible(c.ConstantinopleBlock, newcfg.PetersburgBlock, head) {
			return newCompatError("Petersburg fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
		}
	}
	if isForkIncompatible(c.IstanbulBlock, newcfg.IstanbulBlock, head) {
		return newCompatError("Istanbul fork block", c.IstanbulBlock, newcfg.IstanbulBlock)
	}
	if isForkIncompatible(c.MuirGlacierBlock, newcfg.MuirGlacierBlock, head) {
		return newCompatError("Muir Glacier fork block", c.MuirGlacierBlock, newcfg.MuirGlacierBlock)
	}
	if isForkIncompatible(c.YoloV1Block, newcfg.YoloV1Block, head) {
		return newCompatError("YOLOv1 fork block", c.YoloV1Block, newcfg.YoloV1Block)
	}
	if isForkIncompatible(c.EWASMBlock, newcfg.EWASMBlock, head) {
		return newCompatError("ewasm fork block", c.EWASMBlock, newcfg.EWASMBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                                 *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158               bool
	IsByzantium, IsConstantinople, IsPetersburg, IsIstanbul bool
	IsYoloV1                                                bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
		IsIstanbul:       c.IsIstanbul(num),
		IsYoloV1:         c.IsYoloV1(num),
	}
}
