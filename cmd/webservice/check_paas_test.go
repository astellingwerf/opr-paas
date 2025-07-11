/*
Copyright 2025, Tax Administration of The Netherlands.
Licensed under the EUPL 1.2.
See LICENSE.md for details.
*/

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/belastingdienst/opr-paas-crypttool/pkg/crypt"
	"github.com/belastingdienst/opr-paas/v2/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	paasName = "paasName"
	repoName = "ssh://git@scm/some-repo.git"
)

func TestCheckPaas(t *testing.T) {
	// Allow all origins for testing
	t.Setenv(allowedOriginsEnv, "*")

	// generate private/public keys
	priv, err := os.CreateTemp("", "private")
	require.NoError(t, err, "Creating tempfile for private key")
	defer os.Remove(priv.Name()) // clean up

	pub, err := os.CreateTemp("", "public")
	require.NoError(t, err, "Creating tempfile for public key")
	defer os.Remove(pub.Name()) // clean up

	crypt.GenerateKeyPair(priv.Name(), pub.Name()) //nolint:errcheck // this is fine in test
	getConfig()
	_config.PublicKeyPath = pub.Name()
	_config.PrivateKeyPath = priv.Name()
	assert.Nil(t, _crypt)
	rsa := getRsa(paasName)

	encrypted, err := rsa.Encrypt([]byte("My test string"))
	require.NoError(t, err)

	toBeDecryptedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{
			SSHSecrets: map[string]string{repoName: encrypted},
			Capabilities: v1alpha1.PaasCapabilities{
				"sso": v1alpha1.PaasCapability{
					Enabled:    true,
					SSHSecrets: map[string]string{repoName: encrypted},
				},
			},
		},
	}

	err = CheckPaas(rsa, toBeDecryptedPaas)
	require.NoError(t, err)

	notTeBeDecryptedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{SSHSecrets: map[string]string{repoName: "bm90RGVjcnlwdGFibGU="}},
	}

	// Must be able to decrypt this
	err = CheckPaas(rsa, notTeBeDecryptedPaas)
	require.Error(t, err)

	partialToBeDecryptedPaas := &v1alpha1.Paas{
		ObjectMeta: metav1.ObjectMeta{
			Name: paasName,
		},
		Spec: v1alpha1.PaasSpec{
			SSHSecrets: map[string]string{repoName: encrypted},
			Capabilities: v1alpha1.PaasCapabilities{
				"sso": v1alpha1.PaasCapability{
					Enabled:    true,
					SSHSecrets: map[string]string{repoName: "bm90RGVjcnlwdGFibGU="},
				},
			},
		},
	}

	// Must error as it can be partially decrypted
	err = CheckPaas(rsa, partialToBeDecryptedPaas)
	require.Error(t, err)
}
