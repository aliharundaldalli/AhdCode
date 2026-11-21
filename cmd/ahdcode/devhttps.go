package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ahdcode/internal/backend/golang/ahdruntime"
	"ahdcode/internal/localdev"
)

const devHTTPSURLEnv = "AHDCODE_DEV_HTTPS_URL"

type devHTTPSFiles struct{ cert, key string }

func parseDevArguments(args []string, diagnostics io.Writer) (entry string, https bool, files devHTTPSFiles, err error) {
	flags := flag.NewFlagSet("ahdcode dev", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	flags.BoolVar(&https, "https", false, "serve local HTTPS")
	flags.StringVar(&files.cert, "cert", "", "certificate PEM file")
	flags.StringVar(&files.key, "key", "", "private key PEM file")
	flags.Usage = func() {
		fmt.Fprintln(diagnostics, "Usage: ahdcode dev [--https [--cert FILE --key FILE]] <entry.ahd>")
	}
	if err = flags.Parse(args); err != nil {
		return
	}
	if flags.NArg() != 1 || strings.HasPrefix(flags.Arg(0), "-") {
		err = fmt.Errorf("exactly one entry module is expected, as in: ahdcode dev app.ahd")
		return
	}
	pairSpecified := false
	flags.Visit(func(f *flag.Flag) { pairSpecified = pairSpecified || f.Name == "cert" || f.Name == "key" })
	if pairSpecified && (!https || files.cert == "" || files.key == "") {
		err = fmt.Errorf("--cert and --key must be provided together with --https")
		return
	}
	entry = flags.Arg(0)
	return
}

func prepareDevHTTPS(entry string, files devHTTPSFiles) (devHTTPSFiles, string, error) {
	var ca string
	if files.cert == "" {
		home, err := localdev.Home()
		if err != nil {
			return files, "", err
		}
		if err := devHTTPSPathWithoutLinks(home); err != nil {
			return files, "", err
		}
		dir := filepath.Join(home, "https")
		// Check before generating: a configuration override must not put
		// credentials under the application's public or source tree.
		if err := devHTTPSOutsideProject(entry, filepath.Join(dir, "localhost-key.pem")); err != nil {
			return files, "", err
		}
		if err := checkDevHTTPSPrivacy(home); err != nil {
			return files, "", err
		}
		if err := ensureDevHTTPSStore(home, dir); err != nil {
			return files, "", err
		}
		if err := devHTTPSPathWithoutLinks(dir); err != nil {
			return files, "", err
		}
		if err := checkDevHTTPSPrivacy(dir); err != nil {
			return files, "", err
		}
		files = devHTTPSFiles{filepath.Join(dir, "localhost.pem"), filepath.Join(dir, "localhost-key.pem")}
		ca = filepath.Join(dir, "root-ca.pem")
	}
	cert, err := filepath.Abs(files.cert)
	if err != nil {
		return files, "", fmt.Errorf("could not resolve local HTTPS certificate path")
	}
	key, err := filepath.Abs(files.key)
	if err != nil {
		return files, "", fmt.Errorf("could not resolve local HTTPS private-key path")
	}
	files = devHTTPSFiles{cert, key}
	if err := devHTTPSOutsideProject(entry, key); err != nil {
		return files, "", err
	}
	if err := files.check(ca); err != nil {
		return files, "", err
	}
	return files, ca, nil
}

func (files devHTTPSFiles) check(ca string) error {
	for _, file := range []string{files.cert, files.key} {
		if err := devHTTPSPathWithoutLinks(file); err != nil {
			return err
		}
	}
	certInfo, certErr := os.Stat(files.cert)
	keyInfo, keyErr := os.Stat(files.key)
	if certErr == nil && keyErr == nil && os.SameFile(certInfo, keyInfo) {
		return fmt.Errorf("local HTTPS certificate and private key must be separate files")
	}
	if err := checkDevHTTPSPrivacy(filepath.Dir(files.key)); err != nil {
		return err
	}
	if err := checkDevHTTPSPrivacy(files.key); err != nil {
		return err
	}
	now := time.Now()
	config, err := ahdruntime.AhdHTTPDevLoadTLS(files.cert, files.key, now)
	if err != nil {
		return fmt.Errorf("%w. Supply a valid pair with --cert and --key or move the expired default store before regenerating", err)
	}
	if ca != "" {
		return verifyDevHTTPSCA(ca, config.Certificates[0].Leaf, now)
	}
	return nil
}

func devHTTPSPathWithoutLinks(path string) error {
	if err := ahdruntime.AhdHTTPDevCheckTLSPath(path); err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("local HTTPS credential path is missing or unreadable: %s", path)
	}
	relative, err := filepath.Rel(path, resolved)
	if err != nil || relative != "." {
		return fmt.Errorf("local HTTPS credential paths must not use symbolic links or junctions")
	}
	return nil
}

func devHTTPSOutsideProject(entry, key string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot resolve the local HTTPS working directory")
	}
	cleanKey, err := filepath.Abs(key)
	if err != nil {
		return fmt.Errorf("cannot resolve the local HTTPS private-key path")
	}
	keyCandidates := []string{cleanKey}
	canonicalKey, canonicalErr := devHTTPSCanonicalPath(cleanKey)
	if canonicalErr == nil && canonicalKey != cleanKey {
		keyCandidates = append(keyCandidates, canonicalKey)
	}
	for _, directory := range []string{filepath.Dir(entry), cwd} {
		absDir, err := filepath.Abs(directory)
		if err != nil {
			return fmt.Errorf("cannot resolve the local HTTPS project directory")
		}
		project, err := filepath.EvalSymlinks(absDir)
		if err != nil {
			return fmt.Errorf("cannot resolve the local HTTPS project directory")
		}
		dirCandidates := []string{absDir}
		if project != absDir {
			dirCandidates = append(dirCandidates, project)
		}
		for _, dir := range dirCandidates {
			for _, candidate := range keyCandidates {
				if devHTTPSPathWithin(dir, candidate) {
					return fmt.Errorf("keep the local HTTPS private key outside the project and working directories")
				}
			}
		}
	}
	if canonicalErr != nil {
		return fmt.Errorf("cannot resolve the local HTTPS private-key path")
	}
	return nil
}

// devHTTPSCanonicalPath resolves symlinks in the deepest existing ancestor
// of path and reattaches missing trailing segments. It is used ONLY to reject
// private-key paths inside the project or working directory and must NEVER
// replace the path passed to devHTTPSPathWithoutLinks.
func devHTTPSCanonicalPath(path string) (string, error) {
	cleaned, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	var tail []string
	for current := cleaned; ; {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			parts := make([]string, 0, 1+len(tail))
			parts = append(parts, resolved)
			parts = append(parts, tail...)
			return filepath.Join(parts...), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return cleaned, nil
		}
		tail = append([]string{filepath.Base(current)}, tail...)
		current = parent
	}
}

func devHTTPSPathWithin(dir, candidate string) bool {
	relative, err := filepath.Rel(dir, candidate)
	return err == nil && (relative == "." || filepath.IsLocal(relative))
}

func ensureDevHTTPSStore(home, dir string) error {
	if info, err := os.Lstat(dir); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("the local HTTPS store must be a real directory")
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("cannot inspect the local HTTPS store: %w", err)
	}

	// Generate all three files in a private sibling so another dev process
	// never loads a half-written certificate or an unmatched key.
	stage := filepath.Join(home, ".https-"+rand.Text())
	if err := mkdirPrivateDevHTTPS(stage); err != nil {
		return fmt.Errorf("could not create private local HTTPS storage: %w", err)
	}
	defer os.Remove(stage)
	for _, name := range []string{"localhost.pem", "localhost-key.pem", "root-ca.pem"} {
		defer os.Remove(filepath.Join(stage, name))
	}
	if err := writeDevHTTPSCertificate(stage, time.Now()); err != nil {
		return fmt.Errorf("could not create the local HTTPS certificate: %w", err)
	}
	if err := os.Rename(stage, dir); err != nil {
		// A concurrent first use may have published the complete store.
		if info, statErr := os.Lstat(dir); statErr != nil || !info.IsDir() {
			return fmt.Errorf("could not publish the local HTTPS store: %w", err)
		}
	}
	return nil
}

func verifyDevHTTPSCA(path string, leaf *x509.Certificate, now time.Time) error {
	badCA := fmt.Errorf("the default local HTTPS CA is missing, invalid, or does not issue this certificate; move the default store before regenerating")
	if err := devHTTPSPathWithoutLinks(path); err != nil {
		return badCA
	}
	file, err := os.Open(path)
	if err != nil {
		return badCA
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return badCA
	}
	const limit = 64 << 10
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || len(data) > limit {
		return badCA
	}
	block, rest := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" || strings.TrimSpace(string(rest)) != "" {
		return badCA
	}
	root, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !root.IsCA || root.CheckSignatureFrom(root) != nil {
		return badCA
	}
	pool := x509.NewCertPool()
	pool.AddCert(root)
	if _, err := leaf.Verify(x509.VerifyOptions{
		Roots: pool, CurrentTime: now, DNSName: "localhost",
	}); err != nil {
		return badCA
	}
	return nil
}

func writeDevHTTPSCertificate(dir string, now time.Time) error {
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial := func() (*big.Int, error) {
		n, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
		if err != nil {
			return nil, err
		}
		return n.SetBit(n, 127, 1), nil
	}
	rootSerial, err := serial()
	if err != nil {
		return err
	}
	serverSerial, err := serial()
	if err != nil {
		return err
	}
	ips := []net.IP{net.IPv4(127, 0, 0, 1).To4(), net.IPv6loopback}
	root := &x509.Certificate{
		SerialNumber: rootSerial, Subject: pkix.Name{CommonName: "AhdCode Local Development CA"},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: now.AddDate(0, 0, 45),
		IsCA: true, BasicConstraintsValid: true, MaxPathLenZero: true,
		KeyUsage:                    x509.KeyUsageCertSign,
		PermittedDNSDomainsCritical: true, PermittedDNSDomains: []string{"localhost"},
		PermittedIPRanges: []*net.IPNet{{IP: ips[0], Mask: net.CIDRMask(32, 32)}, {IP: ips[1], Mask: net.CIDRMask(128, 128)}},
	}
	leaf := &x509.Certificate{
		SerialNumber: serverSerial, Subject: pkix.Name{CommonName: "localhost"},
		NotBefore: root.NotBefore, NotAfter: now.AddDate(0, 0, 30),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{"localhost"}, IPAddresses: ips,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, root, root, &rootKey.PublicKey, rootKey)
	if err != nil {
		return err
	}
	// Use the encoded CA's generated subject key identifier to populate the
	// leaf's authority key identifier for clients using strict verification.
	issuer, err := x509.ParseCertificate(rootDER)
	if err != nil {
		return err
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, issuer, &serverKey.PublicKey, rootKey)
	if err != nil {
		return err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	if err != nil {
		return err
	}
	// Persist only the leaf private key. The root signing key is ephemeral.
	for _, file := range []struct {
		name, kind string
		der        []byte
	}{
		{"localhost.pem", "CERTIFICATE", leafDER},
		{"localhost-key.pem", "PRIVATE KEY", privateDER},
		{"root-ca.pem", "CERTIFICATE", rootDER},
	} {
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: file.kind, Bytes: file.der})
		if err := os.WriteFile(filepath.Join(dir, file.name), pemBytes, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func devHTTPSWebURL(environment webEnvironment) (string, error) {
	host := strings.ToLower(environment.serverHost)
	port, err := strconv.Atoi(environment.serverPort)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("local HTTPS needs SERVER_PORT between 1 and 65535")
	}
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		return "", fmt.Errorf("local HTTPS needs SERVER_HOST=localhost, 127.0.0.1, or ::1")
	}
	return "https://" + net.JoinHostPort(host, strconv.Itoa(port)), nil
}
