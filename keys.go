package main

// manage AES keys via KeyCzar / dkeyczar

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	dkeyczar "github.com/dgryski/dkeyczar"
)

type CypherKey struct {
	Mgr *dkeyczar.KeyManager
	Loc string

	crypter dkeyczar.Crypter
}

// convenience method for getting started
func OpenExistingOrCreateNewKey(cfg *Config) (key *CypherKey, err error) {
	if KeyExists(cfg) {
		return LoadKey(cfg)
	}
	return NewKey(cfg)
}

func KeyExists(cfg *Config) bool {
	loc := cfg.KeyLocation()
	if !DirExists(loc) {
		return false
	}
	if !FileExists(loc + "/meta") {
		return false
	}
	if !FileExists(loc + "/1") {
		return false
	}
	return true
}

func NewKey(cfg *Config) (key *CypherKey, err error) {

	key = &CypherKey{}
	km := *MakeKeyMgr()

	key.Loc = cfg.KeyLocation()
	if !DirExists(key.Loc) {
		os.MkdirAll(key.Loc, 0700)
	} else {
		// we already have a key directory, so just make a new key and promote it
		return nil, fmt.Errorf("location '%s' already has a key it", key.Loc)
	}

	updateKeyDir(key.Loc, km, nil)

	// make actual key
	c := loadCrypter("")
	err = loadLocationReader(km, key.Loc, c)
	if err != nil {
		panic(fmt.Sprintf("could not load keyset from location '%s': %s", key.Loc, err))
	}

	status := dkeyczar.S_PRIMARY
	//status := dkeyczar.S_ACTIVE
	//status := dkeyczar.S_INACTIVE

	err = km.AddKey(uint(0), status)
	if err != nil {
		panic(fmt.Sprintf("error adding key: %s", err))
	}
	updateKeyDir(key.Loc, km, c)

	// don't need these if we create with S_PRIMARY, but
	// here's how to promote to primary if needed later.
	if status != dkeyczar.S_PRIMARY {
		keyversion := 1
		km.Promote(keyversion)
		updateKeyDir(key.Loc, km, nil)
	}

	key.InstantiateCrypter()
	return key, nil
}

func (k *CypherKey) DeleteKey() (err error) {
	if DirExists(k.Loc) {
		return os.RemoveAll(k.Loc)
	}
	return nil
}

func MakeKeyMgr() *dkeyczar.KeyManager {
	// make key set
	keytype := dkeyczar.T_AES
	km := dkeyczar.NewKeyManager()
	keypurpose := dkeyczar.P_DECRYPT_AND_ENCRYPT
	km.Create("goq.aes.key", keypurpose, keytype)
	return &km
}

func LoadKey(cfg *Config) (*CypherKey, error) {

	loc := cfg.KeyLocation()
	if !DirExists(loc) {
		return nil, fmt.Errorf("request to LoadKey from bad Loc: %s", loc)
	}

	km := *MakeKeyMgr()
	err := loadLocationReader(km, loc, nil)
	if err != nil {
		return nil, fmt.Errorf("could not load keyset from location '%s': %s", loc, err)
	}

	k := &CypherKey{Mgr: &km, Loc: loc}
	k.InstantiateCrypter()
	return k, nil
}

func (k *CypherKey) InstantiateCrypter() {
	c := loadCrypter("")
	r := loadReader(k.Loc, c)
	if r == nil {
		panic(fmt.Sprintf("could not read keys from '%s'", k.Loc))
	}

	// a crypter can decode as well as encode
	crypter, err := dkeyczar.NewCrypter(r)
	if err != nil {
		panic(err)
	}
	crypter.SetEncoding(dkeyczar.NO_ENCODING)
	//crypter.SetEncoding(dkeyczar.BASE64W)
	k.crypter = crypter
}

func (k *CypherKey) Encrypt(plain []byte) []byte {

	output, err := k.crypter.Encrypt(plain)
	if err != nil {
		panic(err)
	}

	return []byte(output)
}

func (k *CypherKey) Decrypt(cypher []byte) []byte {

	output, err := k.crypter.Decrypt(string(cypher))
	if err != nil {
		panic(err)
	}

	return output
}

func (k *CypherKey) IsValid() bool {
	plain := []byte("hello world")
	cypher := k.Encrypt(plain)
	cs := string(cypher)
	plain2 := string(k.Decrypt(cypher))

	fmt.Printf("plain: '%s', cypher: '%s',  plain2: '%s'\n", plain, cypher, plain2)

	if cs != string(plain) && string(plain) == plain2 {
		return true
	}
	return false
}

func updateKeyDir(location string, km dkeyczar.KeyManager, encrypter dkeyczar.Encrypter) {

	s := km.ToJSONs(encrypter)

	ioutil.WriteFile(location+"/meta", []byte(s[0]), 0600)

	for i := 1; i < len(s); i++ {
		fname := location + "/" + strconv.Itoa(i)
		ioutil.WriteFile(fname, []byte(s[i]), 0600)
	}
}

func loadLocationReader(km dkeyczar.KeyManager, location string, crypter dkeyczar.Crypter) error {
	if location == "" {
		panic("missing required location argument")
	}

	lr := dkeyczar.NewFileReader(location)

	if crypter != nil {
		//fmt.Println("decrypting keys..")
		lr = dkeyczar.NewEncryptedReader(lr, crypter)
	}

	err := km.Load(lr)
	if err != nil {
		return fmt.Errorf("failed to load key: %s", err)
	}
	return nil
}

func loadCrypter(optCrypter string) dkeyczar.Crypter {
	if optCrypter == "" {
		return nil
	}

	//fmt.Println("using crypter:", optCrypter)
	r := dkeyczar.NewFileReader(optCrypter)
	crypter, err := dkeyczar.NewCrypter(r)
	if err != nil {
		panic(fmt.Sprintf("failed to load crypter: %s", err))
		return nil
	}
	return crypter
}

func loadReader(optLocation string, crypter dkeyczar.Crypter) dkeyczar.KeyReader {
	if optLocation == "" {
		panic("missing required location argument")
		return nil
	}

	lr := dkeyczar.NewFileReader(optLocation)

	if crypter != nil {
		//fmt.Println("decrypting keys..")
		lr = dkeyczar.NewEncryptedReader(lr, crypter)
	}

	return lr
}
