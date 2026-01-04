package config

import (
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer/external"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer/key_file"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer/pkcs11"
	"github.com/aliyunidaas/alibaba-cloud-idaas/signer/yubikey_piv"
	"github.com/pkg/errors"
)

func NewExJwtSignerFromConfig(conf *ExSingerConfig) (*signer.ExJwtSigner, error) {
	if conf == nil {
		return nil, errors.New("config is nil")
	}
	//if conf.KeyID == "" {
	//	return nil, errors.New("key id is empty")
	//}
	if conf.Algorithm == "" {
		return nil, errors.New("algorithm is empty")
	}
	jwtSignAlgorithm, parseJwtSignAlgorithmErr := signer.ParseJwtSignAlgorithm(conf.Algorithm)
	if parseJwtSignAlgorithmErr != nil {
		return nil, errors.Wrap(parseJwtSignAlgorithmErr, "parse jwt sign algorithm")
	}

	// should have one set
	var confs []string
	if conf.Pkcs11 != nil {
		confs = append(confs, "pkcs11")
	}
	if conf.YubikeyPiv != nil {
		confs = append(confs, "yubikey_piv")
	}
	if conf.ExternalCommand != nil {
		confs = append(confs, "external_command")
	}
	if conf.KeyFile != nil {
		confs = append(confs, "key_file")
	}
	if len(confs) == 0 {
		return nil, errors.New("requires at least one; pkcs11, yubikey_piv or external_command")
	}
	if len(confs) > 1 {
		return nil, errors.Errorf("only one config may be specified, found: %v", confs)
	}

	var exSigner signer.ExSigner
	var exSignerErr error
	if conf.Pkcs11 != nil {
		exSigner, exSignerErr = NewPkcs11SignerFromConfig(conf.Pkcs11)
	} else if conf.YubikeyPiv != nil {
		exSigner, exSignerErr = NewYubiKeyPivSignerFromConfig(conf.YubikeyPiv)
	} else if conf.ExternalCommand != nil {
		exSigner, exSignerErr = NewExCommandSignerFromConfig(conf.ExternalCommand)
	} else if conf.KeyFile != nil {
		exSigner, exSignerErr = NewKeyFileFromConfig(conf.KeyFile)
	}
	if exSignerErr != nil {
		return nil, errors.Errorf("exsigner initialization error: %s, type: %s", exSignerErr, confs[0])
	}

	exJwtSigner := signer.NewExJwtSigner(conf.KeyID, jwtSignAlgorithm, exSigner)
	return exJwtSigner, nil
}

func NewKeyFileFromConfig(conf *ExSingerKeyFileConfig) (*key_file.KeyFileSigner, error) {
	return key_file.NewKeyFileSigner(conf.Key, conf.File, conf.Password)
}

func NewExCommandSignerFromConfig(conf *ExSignerExternalCommandConfig) (*external.ExCommandSigner, error) {
	return external.NewExCommandSigner(conf.Command, conf.Parameter)
}

func NewPkcs11SignerFromConfig(conf *ExSignerPkcs11Config) (*pkcs11.Pkcs11Signer, error) {
	return pkcs11.NewPkcs11Signer(conf.LibraryPath, conf.TokenLabel, conf.Pin, conf.KeyLabel)
}

func NewYubiKeyPivSignerFromConfig(conf *ExSignerYubikeyPivConfig) (*yubikey_piv.YubiKeyPivSigner, error) {
	return yubikey_piv.NewYubiKeyPivSigner(conf.Slot, conf.Pin, conf.PinPolicy)
}
