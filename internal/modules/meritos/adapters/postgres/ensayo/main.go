package main

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
	merapp "vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

func run() error {
	var c configuration
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 256*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&c); err != nil {
		return errors.New("input_invalid")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("input_trailing")
	}
	defer clear(c.Crypto.Seed)
	defer clear(c.Crypto.HMAC)
	defer func() {
		for _, key := range c.Keys {
			clear(key.HMAC)
		}
	}()
	if c.Mode == "rbac-descriptor" {
		return describeRBAC(c.RBAC)
	}
	if c.Mode == "plan-descriptor" {
		return describePlan(c.Operations)
	}
	if c.Mode == "descriptor" {
		cfg, key, err := trust(c.Crypto)
		if err != nil {
			return err
		}
		defer clear(key)
		spki, err := x509.MarshalPKIXPublicKey(key.Public())
		if err != nil {
			return err
		}
		h := sha256.Sum256(spki)
		configSHA, err := cfg.HuellaSHA256ParaGobierno()
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"spki": spki, "spki_sha256": hex.EncodeToString(h[:]), "configuration_sha256": configSHA})
	}
	if c.Mode != "consume" || len(c.Operations) == 0 || len(c.Operations) > 32 {
		return errors.New("mode_invalid")
	}
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(90*time.Second))
	defer cancel()
	for _, o := range c.Operations {
		a, ok := c.Actors[o.Actor]
		if !ok {
			return errors.New("actor_missing")
		}
		crypto := c.Crypto
		_, audience := merapp.EspecificacionAutorizacion(o.Command.Accion)
		if len(c.Keys) > 0 {
			key, exists := c.Keys[audience]
			if !exists {
				return errors.New("key_audience_missing")
			}
			crypto.KeyID, crypto.KeyVersion, crypto.HMAC, crypto.Issuer = key.ID, key.Version, key.HMAC, key.Issuer
			crypto.GovernmentRevision, crypto.GovernmentSHA, crypto.KeyFrom, crypto.KeyUntil = key.GovernmentRevision, key.GovernmentSHA, key.From, key.Until
		}
		result, err := execute(ctx, crypto, a, o)
		if err != nil {
			return fmt.Errorf("operation %s: %w", o.Name, err)
		}
		if result.Codigo != o.Expected {
			return fmt.Errorf("operation %s: code_mismatch", o.Name)
		}
		if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"name": o.Name, "result": result}); err != nil {
			return err
		}
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
