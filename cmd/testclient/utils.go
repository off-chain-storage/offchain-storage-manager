package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	storagemgrPB "github.com/off-chain-storage/offchain-storage-manager/proto"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

/* utils.go - Generate by AI */

// For Base64
func mustB64(s string) []byte {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		log.WithError(err).Fatal("failed to decode base64")
		panic(err)
	}
	return b
}

// For Hex
func hexToBytes(s string) []byte {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	if i := strings.IndexByte(s, '_'); i >= 0 {
		s = s[:i]
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		log.WithError(err).Fatal("failed to decode hex")
		panic(err)
	}
	return b
}

// For Uint64
func u64(s string) uint64 {
	v, err := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	if err != nil {
		log.WithError(err).Fatal("failed to parse uint64")
		panic(err)
	}
	return v
}

// For Zero Byte
func zeroByte() []byte {
	return []byte{0x00}
}

// Regex
var (
	reDigest = regexp.MustCompile(`digest:\s*([A-Za-z0-9+/=]+)\s*,`)
	reHash   = regexp.MustCompile(`hash:\s*0x([0-9a-fA-F]+)\s*,`)
	reSigR   = regexp.MustCompile(`r:\s*0x([0-9a-fA-F]+)`)
	reSigS   = regexp.MustCompile(`s:\s*0x([0-9a-fA-F]+)`)
	reOdd    = regexp.MustCompile(`odd_y_parity:\s*(true|false)`)

	reChain    = regexp.MustCompile(`chain_id:\s*Some\(\s*([0-9]+)\s*,?\s*\)`)
	reNonce    = regexp.MustCompile(`nonce:\s*([0-9]+)\s*,`)
	reGPHex    = regexp.MustCompile(`gas_price:\s*0x([0-9a-fA-F]+)\s*,`)
	reGPNum    = regexp.MustCompile(`gas_price:\s*([0-9]+)\s*,`)
	reGLimit   = regexp.MustCompile(`gas_limit:\s*([0-9]+)\s*,`)
	reToCreate = regexp.MustCompile(`to:\s*Create`)
	reToCall   = regexp.MustCompile(`to:\s*Call\(\s*0x([0-9a-fA-F]+)\s*,?\s*\)`)
	reValue    = regexp.MustCompile(`value:\s*TxValue\(\s*(0x[0-9a-fA-F]+(?:_[A-Za-z0-9]+)?)\s*,?\s*\)`)
	reInput    = regexp.MustCompile(`input:\s*0x([0-9a-fA-F]+)\s*,?`)
)

func extractBlocks(src, marker string) ([]string, error) {
	var blocks []string
	idx := 0
	for {
		pos := strings.Index(src[idx:], marker)
		if pos < 0 {
			break
		}
		pos += idx
		start := pos + len(marker)
		block, next, err := readBlock(src, start)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
		idx = next
	}
	return blocks, nil
}

func readBlock(src string, start int) (string, int, error) {
	depth := 1
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start:i], i + 1, nil
			}
		}
	}
	return "", len(src), errors.New("unterminated block starting at " + strconv.Itoa(start))
}

func parseDebugTextToProto(txt string) (*storagemgrPB.ExecutableConsensusOutput, error) {
	if i := strings.Index(txt, "ExecutableConsensusOutput"); i >= 0 {
		txt = txt[i:]
	}

	batchBodies, err := extractBlocks(txt, "ExecutableEthereumBatch {")
	if err != nil {
		log.WithError(err).Fatal("failed to extract blocks")
		return nil, err
	}
	if len(batchBodies) == 0 {
		log.Fatal("no ExecutableEthereumBatch found")
		return nil, errors.New("no ExecutableEthereumBatch found")
	}

	out := &storagemgrPB.ExecutableConsensusOutput{}
	for _, body := range batchBodies {
		dm := reDigest.FindStringSubmatch(body)
		if dm == nil {
			log.Fatal("digest missing")
			return nil, errors.New("digest missing")
		}

		batch := &storagemgrPB.ExecutableEthereumBatch{
			Digest: mustB64(dm[1]),
		}

		txBodies, err := extractBlocks(body, "TransactionSigned {")
		if err != nil {
			log.WithError(err).Fatal("failed to extract blocks")
			return nil, err
		}
		if len(txBodies) == 0 {
			log.Fatal("no TransactionSigned block found")
			return nil, errors.New("no TransactionSigned block found")
		}

		for _, tb := range txBodies {
			hm := reHash.FindStringSubmatch(tb)
			if hm == nil {
				log.Fatal("hash missing")
				return nil, errors.New("hash missing")
			}
			rm := reSigR.FindStringSubmatch(tb)
			if rm == nil {
				log.Fatal("sig.r missing")
				return nil, errors.New("sig.r missing")
			}
			sm := reSigS.FindStringSubmatch(tb)
			if sm == nil {
				log.Fatal("sig.s missing")
				return nil, errors.New("sig.s missing")
			}
			om := reOdd.FindStringSubmatch(tb)
			if om == nil {
				log.Fatal("odd_y_parity missing")
				return nil, errors.New("odd_y_parity missing")
			}

			legacyBlocks, err := extractBlocks(tb, "TxLegacy {")
			if err != nil {
				log.WithError(err).Fatal("failed to extract blocks")
				return nil, err
			}
			if len(legacyBlocks) == 0 {
				log.Fatal("TxLegacy block missing")
				return nil, errors.New("TxLegacy block missing")
			}
			lb := legacyBlocks[0]

			var chain *wrapperspb.UInt64Value
			if cm := reChain.FindStringSubmatch(lb); cm != nil {
				chain = wrapperspb.UInt64(u64(cm[1]))
			}
			nm := reNonce.FindStringSubmatch(lb)
			if nm == nil {
				log.Fatal("nonce missing")
				return nil, errors.New("nonce missing")
			}

			var gp []byte
			if gph := reGPHex.FindStringSubmatch(lb); gph != nil {
				gp = hexToBytes(gph[1])
			} else if gpn := reGPNum.FindStringSubmatch(lb); gpn != nil {
				if strings.TrimSpace(gpn[1]) == "0" {
					gp = zeroByte()
				} else {
					gp = hexToBytes(fmt.Sprintf("%x", u64(gpn[1])))
				}
			} else {
				gp = zeroByte()
			}

			glm := reGLimit.FindStringSubmatch(lb)
			if glm == nil {
				log.Fatal("gas_limit missing")
				return nil, errors.New("gas_limit missing")
			}

			var to *storagemgrPB.To
			switch {
			case reToCreate.FindStringSubmatch(lb) != nil:
				to = &storagemgrPB.To{Kind: &storagemgrPB.To_Create{Create: true}}
			case reToCall.FindStringSubmatch(lb) != nil:
				tc := reToCall.FindStringSubmatch(lb)[1]
				to = &storagemgrPB.To{Kind: &storagemgrPB.To_Call{Call: hexToBytes(tc)}}
			default:
				to = &storagemgrPB.To{Kind: &storagemgrPB.To_Create{Create: true}}
			}

			vm := reValue.FindStringSubmatch(lb)
			if vm == nil {
				log.Fatal("value missing")
				return nil, errors.New("value missing")
			}
			im := reInput.FindStringSubmatch(lb)
			if im == nil {
				log.Fatal("input missing")
				return nil, errors.New("input missing")
			}

			legacy := &storagemgrPB.TxLegacy{
				ChainId:  chain,
				Nonce:    u64(nm[1]),
				GasPrice: gp,
				GasLimit: u64(glm[1]),
				To:       to,
				Value:    hexToBytes(vm[1]),
				Input:    hexToBytes(im[1]),
			}

			batch.Data = append(batch.Data, &storagemgrPB.TransactionSigned{
				Hash: hexToBytes(hm[1]),
				Signature: &storagemgrPB.Signature{
					R:          hexToBytes(rm[1]),
					S:          hexToBytes(sm[1]),
					OddYParity: om[1] == "true",
				},
				Tx: &storagemgrPB.TransactionSigned_Legacy{Legacy: legacy},
			})
		}

		out.Data = append(out.Data, batch)
	}
	return out, nil
}

func parseLog(in string, outPath string, verify bool) (*storagemgrPB.ExecutableConsensusOutput, error) {
	b, err := os.ReadFile(in)
	if err != nil {
		log.WithError(err).Fatal("failed to read file")
		return nil, err
	}

	req, err := parseDebugTextToProto(string(b))
	if err != nil {
		log.WithError(err).Fatal("failed to parse debug text to proto")
		return nil, err
	}

	// proto → JSON (proto3 JSON 규칙)
	m := protojson.MarshalOptions{
		UseProtoNames:   true, // snake_case 유지
		EmitUnpopulated: true,
		Indent:          "  ",
	}
	j, err := m.Marshal(req)
	if err != nil {
		log.WithError(err).Fatal("failed to marshal proto to json")
		return nil, err
	}

	if err := os.WriteFile(outPath, j, 0644); err != nil {
		log.WithError(err).Fatal("failed to write file")
		return nil, err
	}
	log.Infof("wrote %s (%d bytes)", outPath, len(j))

	if verify {
		var back storagemgrPB.ExecutableConsensusOutput
		u := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := u.Unmarshal(j, &back); err != nil {
			log.WithError(err).Fatal("failed to unmarshal json to proto")
			return nil, err
		}
		log.Infof("verify OK: batches=%d, txs_in_first=%d",
			len(back.GetData()),
			func() int {
				if len(back.GetData()) > 0 {
					return len(back.GetData()[0].GetData())
				}
				return 0
			}(),
		)
	}

	return req, nil
}
