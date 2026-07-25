// Command sign generates the release signing keypair and signs release
// artifacts. CI calls it with the secret key in an environment variable; a
// maintainer calls -genkey once to create the key.
//
//	go run ./scripts/sign -genkey -out ~/.reeve/release-key
//	REEVE_SIGNING_KEY="$(cat key)" go run ./scripts/sign -comment v1.2.3 dist/agent-linux-amd64
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/thehelvijs/Reeve/signing"
)

// pubKeyPath is where the agent expects the committed public key.
const pubKeyPath = "agent/release_pubkey.txt"

func main() {
	log.SetFlags(0)
	genkey := flag.Bool("genkey", false, "generate a signing keypair")
	out := flag.String("out", "", "with -genkey: where to write the secret key")
	pubOut := flag.String("pub-out", pubKeyPath, "with -genkey: where to write the public key")
	comment := flag.String("comment", "", "trusted comment to embed in each signature (e.g. the version)")
	keyFile := flag.String("key", "", "secret key file (defaults to $REEVE_SIGNING_KEY)")
	flag.Parse()

	if *genkey {
		if err := generate(*out, *pubOut); err != nil {
			log.Fatalf("sign: %v", err)
		}
		return
	}
	if flag.NArg() == 0 {
		log.Fatal("sign: pass one or more files to sign, or -genkey")
	}
	sk, err := loadSecretKey(*keyFile)
	if err != nil {
		log.Fatalf("sign: %v", err)
	}
	for _, path := range flag.Args() {
		sig, err := signing.SignFile(sk, path, *comment)
		if err != nil {
			log.Fatalf("sign %s: %v", path, err)
		}
		if err := os.WriteFile(path+".minisig", []byte(sig), 0o644); err != nil {
			log.Fatalf("sign %s: %v", path, err)
		}
		fmt.Println("signed", path)
	}
}

// generate writes a new keypair, refusing to overwrite an existing secret key:
// losing it means every deployed agent stops trusting new releases.
func generate(secretPath, publicPath string) error {
	if secretPath == "" {
		return fmt.Errorf("-genkey needs -out <path for the secret key>")
	}
	if _, err := os.Stat(secretPath); err == nil {
		return fmt.Errorf("%s already exists; refusing to overwrite a signing key", secretPath)
	}
	sk, pub, err := signing.GenerateKey()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(secretPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(secretPath, []byte(signing.EncodeSecretKey(sk)), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(publicPath, []byte(signing.EncodePublicKey(pub)), 0o644); err != nil {
		return err
	}
	fmt.Printf("secret key: %s (keep it out of the repo, add it to CI as REEVE_SIGNING_KEY)\n", secretPath)
	fmt.Printf("public key: %s (commit this; agents verify against it)\n", publicPath)
	fmt.Print(signing.EncodePublicKey(pub))
	return nil
}

// loadSecretKey reads the key from a file or from REEVE_SIGNING_KEY.
func loadSecretKey(path string) (signing.SecretKey, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return signing.SecretKey{}, err
		}
		return signing.ParseSecretKey(string(data))
	}
	env := os.Getenv("REEVE_SIGNING_KEY")
	if env == "" {
		return signing.SecretKey{}, fmt.Errorf("no signing key: pass -key or set REEVE_SIGNING_KEY")
	}
	return signing.ParseSecretKey(env)
}
