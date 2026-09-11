// Copyright (c) 2026 Aiven, Helsinki, Finland. https://aiven.io/

//go:build kafka

package tests

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"slices"
	"testing"
	"time"

	"github.com/aiven/go-client-codegen/handler/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/aiven/aiven-operator/api/v1alpha1"
	kafkauserconfig "github.com/aiven/aiven-operator/api/v1alpha1/userconfig/service/kafka"
)

func TestServiceUserRotationKafka(t *testing.T) {
	t.Parallel()
	acquireCtx, cancelAcquire := testCtx()
	kafka, release, err := sharedResources.AcquireKafka(acquireCtx)
	cancelAcquire()
	require.NoError(t, err)
	t.Cleanup(release)
	enableRotationKafkaSASL(t, kafka)
	s := newRotationTest(t, kafka.Name)
	first := s.create()
	waitRotationKafka(t, s, first)
	waitRotationKafkaSASL(t, s, first, "authenticated")
	previous := first

	for _, username := range s.rotation.Spec.Usernames[1:] {
		current := s.rotate(previous, username)
		waitRotationKafka(t, s, current)
		waitRotationKafka(t, s, previous)
		waitRotationKafkaSASL(t, s, current, "authenticated")
		waitRotationKafkaSASL(t, s, previous, "authenticated")
		previous = current
	}

	current := s.rotate(previous, s.rotation.Spec.Usernames[0])
	waitRotationKafka(t, s, current)
	waitRotationKafka(t, s, previous)
	waitRotationKafkaSASL(t, s, current, "authenticated")
	waitRotationKafkaSASL(t, s, previous, "authenticated")
	require.False(t, bytes.Equal(first.secret.Data[s.prefix+"PASSWORD"], current.secret.Data[s.prefix+"PASSWORD"]),
		"reusing a Kafka username must give it a new password")
	waitRotationKafkaSASL(t, s, first, "password rejected")
}

func enableRotationKafkaSASL(t *testing.T, kafka *v1alpha1.Kafka) {
	t.Helper()
	ctx, cancel := testCtx()
	defer cancel()
	current := &v1alpha1.Kafka{}
	require.NoError(t, k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), current))
	orig := current.Spec.UserConfig.DeepCopy()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
		defer cancel()
		assert.NoError(t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
			cr := &v1alpha1.Kafka{}
			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), cr); err != nil {
				return err
			}
			cr.Spec.UserConfig = orig
			return k8sClient.Update(ctx, cr)
		}))
	})
	require.NoError(t, retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr := &v1alpha1.Kafka{}
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(kafka), cr); err != nil {
			return err
		}
		if cr.Spec.UserConfig == nil {
			cr.Spec.UserConfig = &kafkauserconfig.KafkaUserConfig{}
		}
		cr.Spec.UserConfig.KafkaAuthenticationMethods = &kafkauserconfig.KafkaAuthenticationMethods{
			Certificate: new(true), Sasl: new(true),
		}
		if cr.Spec.UserConfig.KafkaSaslMechanisms == nil {
			cr.Spec.UserConfig.KafkaSaslMechanisms = &kafkauserconfig.KafkaSaslMechanisms{}
		}
		cr.Spec.UserConfig.KafkaSaslMechanisms.Plain = new(true)
		return k8sClient.Update(ctx, cr)
	}))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		attemptCtx, cancelAttempt := context.WithTimeout(ctx, 10*time.Second)
		defer cancelAttempt()
		svc, err := avnGen.ServiceGet(attemptCtx, kafka.Spec.Project, kafka.Name)
		require.NoError(c, err)
		assert.Contains(c, serviceRunningStatesAiven, svc.State)
		methods, ok := svc.UserConfig["kafka_authentication_methods"].(map[string]any)
		require.True(c, ok)
		assert.Equal(c, true, methods["sasl"])
		assert.Equal(c, true, methods["certificate"])
		mechanisms, ok := svc.UserConfig["kafka_sasl_mechanisms"].(map[string]any)
		require.True(c, ok)
		assert.Equal(c, true, mechanisms["plain"])
		assert.True(c, slices.ContainsFunc(svc.Components, func(component service.ComponentOut) bool {
			return component.Component == "kafka" && component.KafkaAuthenticationMethod == service.KafkaAuthenticationMethodTypeSasl
		}), "Kafka must expose a SASL endpoint")
	}, waitRunningTimeout, rotationPoll)
}

func waitRotationKafkaSASL(t *testing.T, s *rotationTest, publication *rotationPublication, expected string) {
	t.Helper()
	data := publication.secret.Data
	for _, suffix := range []string{"SASL_HOST", "SASL_PORT", "USERNAME", "PASSWORD", "CA_CERT"} {
		require.NotEmpty(t, data[s.prefix+suffix], "missing connection field %s", suffix)
	}
	ca := x509.NewCertPool()
	require.True(t, ca.AppendCertsFromPEM(data[s.prefix+"CA_CERT"]), "the published CA must be valid PEM")
	address := net.JoinHostPort(string(data[s.prefix+"SASL_HOST"]), string(data[s.prefix+"SASL_PORT"]))
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		dialer := tls.Dialer{Config: &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: ca}}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		require.NoError(c, err)
		defer conn.Close()
		deadline, ok := ctx.Deadline()
		require.True(c, ok)
		require.NoError(c, conn.SetDeadline(deadline))

		// SaslHandshake v1 selects PLAIN; SaslAuthenticate v0 checks the password.
		// https://kafka.apache.org/39/design/protocol/#The_Messages_SaslAuthenticate
		require.Zero(c, rotationKafkaRequest(c, conn, 17, 1, append([]byte{0, 5}, "PLAIN"...)))
		auth := []byte("\x00" + string(data[s.prefix+"USERNAME"]) + "\x00" + string(data[s.prefix+"PASSWORD"]))
		payload := append(binary.BigEndian.AppendUint32(nil, uint32(len(auth))), auth...)
		code := rotationKafkaRequest(c, conn, 36, 0, payload)
		outcome := fmt.Sprintf("Kafka error %d", code)
		switch code {
		case 0:
			outcome = "authenticated"
		case 58: // SASL_AUTHENTICATION_FAILED
			outcome = "password rejected"
		}
		assert.Equal(c, expected, outcome)
	}, time.Minute, rotationPoll, "the published Kafka password must determine SASL authentication")
}

// rotationKafkaRequest reads the error code shared by the non-flexible SASL responses.
// Discard the remaining response without logging it: authentication errors may contain credentials.
func rotationKafkaRequest(c *assert.CollectT, conn net.Conn, apiKey, version int16, payload []byte) int16 {
	request := struct {
		Length         int32
		APIKey         int16
		Version        int16
		CorrelationID  int32
		ClientIDLength int16
	}{APIKey: apiKey, Version: version, CorrelationID: int32(apiKey)}
	request.Length = int32(binary.Size(request) - 4 + len(payload))
	var frame bytes.Buffer
	require.NoError(c, binary.Write(&frame, binary.BigEndian, request))
	frame.Write(payload)
	_, err := frame.WriteTo(conn)
	require.NoError(c, err)
	var response struct {
		Length        int32
		CorrelationID int32
		ErrorCode     int16
	}
	require.NoError(c, binary.Read(conn, binary.BigEndian, &response))
	require.Equal(c, request.CorrelationID, response.CorrelationID)
	require.GreaterOrEqual(c, response.Length, int32(6))
	_, err = io.CopyN(io.Discard, conn, int64(response.Length)-6)
	require.NoError(c, err)
	return response.ErrorCode
}

func waitRotationKafka(t *testing.T, s *rotationTest, publication *rotationPublication) {
	t.Helper()
	data := publication.secret.Data
	cert, err := tls.X509KeyPair(data[s.prefix+"ACCESS_CERT"], data[s.prefix+"ACCESS_KEY"])
	require.NoError(t, err)
	ca := x509.NewCertPool()
	require.True(t, ca.AppendCertsFromPEM(data[s.prefix+"CA_CERT"]), "the published CA must be valid PEM")
	address := net.JoinHostPort(string(data[s.prefix+"HOST"]), string(data[s.prefix+"PORT"]))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
		defer cancel()
		user, err := avnGen.ServiceUserGet(ctx, s.rotation.Spec.Project, s.rotation.Spec.ServiceName,
			string(data[s.prefix+"USERNAME"]), service.ServiceUserGetIncludeSecrets(true))
		require.NoError(c, err)
		require.NotNil(c, user.AccessCert)
		require.True(c, bytes.Equal([]byte(*user.AccessCert), data[s.prefix+"ACCESS_CERT"]),
			"the published certificate must belong to the published user")

		requestedCertificate := false
		dialer := tls.Dialer{Config: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    ca,
			GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
				requestedCertificate = true
				return &cert, nil
			},
		}}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		require.NoError(c, err)
		defer conn.Close()
		// ApiVersions is also available before SASL authentication. Require mTLS.
		require.True(c, requestedCertificate, "the broker must request the published client certificate")
		deadline, ok := ctx.Deadline()
		require.True(c, ok)
		require.NoError(c, conn.SetDeadline(deadline))

		// ApiVersions v0 needs no topic ACLs. A protocol response confirms that
		// the broker accepted the TLS connection, including TLS 1.3's final flight.
		// https://kafka.apache.org/34/design/protocol/#The_Messages_ApiVersions
		request := struct {
			Length         int32
			APIKey         int16
			Version        int16
			CorrelationID  int32
			ClientIDLength int16
		}{APIKey: 18, CorrelationID: 1}
		request.Length = int32(binary.Size(request) - 4)
		require.NoError(c, binary.Write(conn, binary.BigEndian, request))
		var response struct {
			Length        int32
			CorrelationID int32
			ErrorCode     int16
			APICount      int32
		}
		require.NoError(c, binary.Read(conn, binary.BigEndian, &response))
		assert.Equal(c, request.CorrelationID, response.CorrelationID)
		assert.Zero(c, response.ErrorCode)
		assert.Positive(c, response.APICount)
		assert.Equal(c, int32(10)+6*response.APICount, response.Length)
	}, time.Minute, rotationPoll, "the published Kafka certificate and key must authenticate a fresh connection")
}
