package uploads

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Storage interface {
	Save(objectKey string, data []byte, contentType string) (string, error)
	Delete(objectKey string) error
}

func NewStorage() (Storage, error) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("AVATAR_STORAGE_DRIVER")))
	if driver == "" || driver == "local" {
		return localStorage{}, nil
	}
	if driver == "oci" {
		return newOCIStorage()
	}

	return nil, fmt.Errorf("storage de avatar nao suportado: %s", driver)
}

type localStorage struct{}

func (localStorage) Save(objectKey string, data []byte, _ string) (string, error) {
	targetPath := filepath.Join(Dir(), filepath.FromSlash(objectKey))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return "", err
	}

	return PublicURL(strings.Split(objectKey, "/")...), nil
}

func (localStorage) Delete(objectKey string) error {
	err := os.Remove(filepath.Join(Dir(), filepath.FromSlash(objectKey)))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type ociStorage struct {
	namespace string
	bucket    string
	region    string
	accessKey string
	secretKey string
	client    *http.Client
}

func newOCIStorage() (*ociStorage, error) {
	storage := &ociStorage{
		namespace: strings.TrimSpace(os.Getenv("OCI_NAMESPACE")),
		bucket:    strings.TrimSpace(os.Getenv("OCI_BUCKET")),
		region:    strings.TrimSpace(os.Getenv("OCI_REGION")),
		accessKey: strings.TrimSpace(os.Getenv("OCI_ACCESS_KEY")),
		secretKey: strings.TrimSpace(os.Getenv("OCI_SECRET_KEY")),
		client:    &http.Client{Timeout: 20 * time.Second},
	}
	if storage.region == "" {
		storage.region = "sa-saopaulo-1"
	}

	missing := make([]string, 0)
	if storage.namespace == "" {
		missing = append(missing, "OCI_NAMESPACE")
	}
	if storage.bucket == "" {
		missing = append(missing, "OCI_BUCKET")
	}
	if storage.accessKey == "" {
		missing = append(missing, "OCI_ACCESS_KEY")
	}
	if storage.secretKey == "" {
		missing = append(missing, "OCI_SECRET_KEY")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("variaveis do OCI Storage ausentes: %s", strings.Join(missing, ", "))
	}

	return storage, nil
}

func (s *ociStorage) Save(objectKey string, data []byte, contentType string) (string, error) {
	endpoint := s.objectEndpoint(objectKey)
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-content-sha256", sha256Hex(data))

	if err := s.sign(req, data); err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("erro ao enviar foto para OCI Storage: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return s.publicURL(objectKey), nil
}

func (s *ociStorage) Delete(objectKey string) error {
	req, err := http.NewRequest(http.MethodDelete, s.objectEndpoint(objectKey), nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-amz-content-sha256", sha256Hex(nil))

	if err := s.sign(req, nil); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("erro ao remover foto da OCI Storage: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func (s *ociStorage) objectEndpoint(objectKey string) string {
	escapedKey := strings.ReplaceAll(url.PathEscape(objectKey), "%2F", "/")
	return fmt.Sprintf("https://%s.compat.objectstorage.%s.oraclecloud.com/%s/%s", s.namespace, s.region, s.bucket, escapedKey)
}

func (s *ociStorage) publicURL(objectKey string) string {
	if baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OCI_PUBLIC_BASE_URL")), "/"); baseURL != "" {
		return baseURL + "/" + strings.ReplaceAll(url.PathEscape(objectKey), "%2F", "/")
	}

	escapedKey := strings.ReplaceAll(url.PathEscape(objectKey), "%2F", "/")
	return fmt.Sprintf("https://objectstorage.%s.oraclecloud.com/n/%s/b/%s/o/%s", s.region, s.namespace, s.bucket, escapedKey)
}

func (s *ociStorage) sign(req *http.Request, payload []byte) error {
	if req == nil || req.URL == nil {
		return errors.New("request invalida para assinatura")
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("Host", req.URL.Host)

	payloadHash := req.Header.Get("x-amz-content-sha256")
	if payloadHash == "" {
		payloadHash = sha256Hex(payload)
		req.Header.Set("x-amz-content-sha256", payloadHash)
	}

	signedHeaders, canonicalHeaders := canonicalHeaders(req.Header)
	canonicalRequest := strings.Join([]string{
		req.Method,
		uriEncodePath(req.URL.EscapedPath()),
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := strings.Join([]string{dateStamp, s.region, "s3", "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := sigV4SigningKey(s.secretKey, dateStamp, s.region, "s3")
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		s.accessKey,
		credentialScope,
		signedHeaders,
		signature,
	)
	req.Header.Set("Authorization", authorization)

	return nil
}

func canonicalHeaders(headers http.Header) (string, string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		values := headers.Values(key)
		for i := range values {
			values[i] = strings.Join(strings.Fields(values[i]), " ")
		}
		lines = append(lines, key+":"+strings.Join(values, ","))
	}

	return strings.Join(keys, ";"), strings.Join(lines, "\n") + "\n"
}

func uriEncodePath(escapedPath string) string {
	if escapedPath == "" {
		return "/"
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(escapedPath, "/"))
	if strings.HasSuffix(escapedPath, "/") && !strings.HasSuffix(cleaned, "/") {
		cleaned += "/"
	}
	return cleaned
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sigV4SigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
