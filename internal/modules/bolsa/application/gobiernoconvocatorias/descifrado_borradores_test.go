package gobiernoconvocatorias

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func escenarioDescifradoBorradorPrueba(
	t *testing.T,
) (SolicitudDescifradoBorradorDurable, []byte) {
	t.Helper()
	e := nuevoEscenario(t, confirmarBien, 3, 2)
	if _, err := e.servicio.Crear(context.Background(), e.orden); err != nil {
		t.Fatal(err)
	}
	confirmacion := *e.confirmador.ultima
	estado, err := puertosbolsa.EstadoVersionConvocatoria(confirmacion.Version)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := NuevaSolicitudDescifradoBorradorDurable(
		estado, confirmacion.Cifrado.AAD, confirmacion.PerfilCifrado,
		confirmacion.Cifrado.EnvolturaClave, confirmacion.Cifrado.SobreCifrado,
		confirmacion.Cifrado.AtestacionKMS, confirmacion.Procedencia,
	)
	if err != nil {
		t.Fatal(err)
	}
	claro, err := confirmacion.Version.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, claro
}

func TestDescifradoDurableRehidrataSoloVersionCanonicaExacta(t *testing.T) {
	solicitud, claro := escenarioDescifradoBorradorPrueba(t)
	resultado, err := NuevoResultadoDescifradoBorradorDurable(solicitud, claro)
	if err != nil {
		t.Fatal(err)
	}
	recuperada, err := resultado.VersionConvocatoria()
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := recuperada.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, claro) {
		t.Fatalf("agregado recuperado distinto: %v", err)
	}
	recuperada.Contenido.Categorias[0] = "categoria_alterada"
	segunda, err := resultado.VersionConvocatoria()
	if err != nil {
		t.Fatal(err)
	}
	canonicoDefensivo, err := segunda.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonicoDefensivo, claro) {
		t.Fatalf("el retorno compartió memoria con el resultado: %v", err)
	}

	noCanonico := append(append([]byte(nil), claro...), '\n')
	if _, err := NuevoResultadoDescifradoBorradorDurable(solicitud, noCanonico); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("se aceptó claro JSON no canónico: %v", err)
	}
	duplicado := append([]byte(`{"esquema":"bolsa.version-convocatoria.estado.v2",`), claro[1:]...)
	if _, err := NuevoResultadoDescifradoBorradorDurable(solicitud, duplicado); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("se aceptó claro con clave JSON duplicada: %v", err)
	}

	e := nuevoEscenario(t, confirmarBien, 3, 2)
	ajeno, err := e.inicial.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NuevoResultadoDescifradoBorradorDurable(solicitud, ajeno); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("se aceptó agregado de otra referencia: %v", err)
	}
}

func TestRestaurarAADExigeBytesCanonicosYHuellaExacta(t *testing.T) {
	solicitud, _ := escenarioDescifradoBorradorPrueba(t)
	aad, err := solicitud.AAD.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := solicitud.AAD.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	restaurada, err := RestaurarAADCanonicaCifradoBorrador(aad, huella)
	if err != nil || !aadCoincide(restaurada, solicitud.AAD) {
		t.Fatalf("AAD durable no se restauró: %v", err)
	}

	alterada := append(append([]byte(nil), aad...), ' ')
	huellaAlterada := sha256.Sum256(alterada)
	casos := []struct {
		nombre, huella string
	}{
		{"huella original", huella},
		{"huella recalculada sobre JSON no canónico", hex.EncodeToString(huellaAlterada[:])},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := RestaurarAADCanonicaCifradoBorrador(alterada, caso.huella); !errors.Is(err, ErrDescifradoBorradorInvalido) {
				t.Fatalf("AAD alterada aceptada: %v", err)
			}
		})
	}
	duplicada := append([]byte(`{"esquema":"bolsa.convocatoria.borrador.aad.v1",`), aad[1:]...)
	huellaDuplicada := sha256.Sum256(duplicada)
	if _, err := RestaurarAADCanonicaCifradoBorrador(
		duplicada, hex.EncodeToString(huellaDuplicada[:]),
	); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("AAD con clave JSON duplicada aceptada: %v", err)
	}
}

func TestSolicitudDescifradoLigaSobrePerfilProcedenciaYEstado(t *testing.T) {
	original, _ := escenarioDescifradoBorradorPrueba(t)
	perfilAjeno, err := NuevoPerfilCifradoBorrador(
		"perfil:cifrado:borradores:ajeno", 1, huellaHexPrueba('0'), "A256GCM", "A256KW",
	)
	if err != nil {
		t.Fatal(err)
	}
	procedenciaAjena, err := NuevaProcedenciaActoBorrador(
		"pruebas-ajenas", AutoridadActoAutoritativa, "proveedor-pruebas-ajeno", true,
	)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*SolicitudDescifradoBorradorDurable){
		"referencia": func(s *SolicitudDescifradoBorradorDurable) {
			s.EstadoEsperado.Referencia = "proceso:bolsa:referencia-ajena#1"
		},
		"revision": func(s *SolicitudDescifradoBorradorDurable) {
			s.EstadoEsperado.Revision++
		},
		"huella": func(s *SolicitudDescifradoBorradorDurable) {
			s.EstadoEsperado.HuellaEstadoSHA256 = huellaHexPrueba('0')
		},
		"AAD": func(s *SolicitudDescifradoBorradorDurable) {
			material, valida := decodificarMaterialAADCanonicaBorrador(s.AAD.representacion)
			if !valida {
				t.Fatal("AAD de referencia inválida")
			}
			material.HuellaMaterialSHA256 = huellaHexPrueba('0')
			bytesAAD, err := json.Marshal(material)
			if err != nil {
				t.Fatal(err)
			}
			huella := sha256.Sum256(bytesAAD)
			s.AAD, err = RestaurarAADCanonicaCifradoBorrador(bytesAAD, hex.EncodeToString(huella[:]))
			if err != nil {
				t.Fatal(err)
			}
		},
		"perfil":      func(s *SolicitudDescifradoBorradorDurable) { s.PerfilEsperado = perfilAjeno },
		"procedencia": func(s *SolicitudDescifradoBorradorDurable) { s.Procedencia = procedenciaAjena },
		"clave envuelta": func(s *SolicitudDescifradoBorradorDurable) {
			perfil, claveRef, version, material, huellaAAD, _, err :=
				s.EnvolturaClave.DatosParaPersistencia()
			if err != nil {
				t.Fatal(err)
			}
			material[0] ^= 0x80
			s.EnvolturaClave, err = NuevaEnvolturaClaveKMSBorrador(
				perfil, claveRef, version, material, huellaAAD,
			)
			if err != nil {
				t.Fatal(err)
			}
		},
		"nonce": func(s *SolicitudDescifradoBorradorDurable) {
			perfil, nonce, cifrado, huellaAAD, _, err := s.SobreCifrado.DatosParaPersistencia()
			if err != nil {
				t.Fatal(err)
			}
			nonce[0] ^= 0x01
			s.SobreCifrado, err = NuevoSobreCifradoAEADBorrador(perfil, nonce, cifrado, huellaAAD)
			if err != nil {
				t.Fatal(err)
			}
		},
		"texto cifrado": func(s *SolicitudDescifradoBorradorDurable) {
			perfil, nonce, cifrado, huellaAAD, _, err := s.SobreCifrado.DatosParaPersistencia()
			if err != nil {
				t.Fatal(err)
			}
			cifrado[len(cifrado)-1] ^= 0x01
			s.SobreCifrado, err = NuevoSobreCifradoAEADBorrador(perfil, nonce, cifrado, huellaAAD)
			if err != nil {
				t.Fatal(err)
			}
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			alterada := original
			mutar(&alterada)
			if !errors.Is(alterada.Validar(), ErrDescifradoBorradorInvalido) {
				t.Fatal("la solicitud durable alterada no falló cerrada")
			}
		})
	}
}

func TestMaterialDescifradoDevuelveCopiasDefensivas(t *testing.T) {
	solicitud, _ := escenarioDescifradoBorradorPrueba(t)
	aad, envuelta, nonce, cifrado, err := solicitud.MaterialParaConectorConfiable()
	if err != nil {
		t.Fatal(err)
	}
	aad[0] ^= 0x01
	envuelta[0] ^= 0x01
	nonce[0] ^= 0x01
	cifrado[0] ^= 0x01
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("las copias expuestas alteraron la solicitud: %v", err)
	}
}

func TestSobreYEnvolturaPersistidosExigenEsquemaYHuella(t *testing.T) {
	solicitud, _ := escenarioDescifradoBorradorPrueba(t)
	esquemaEnvoltura, err := solicitud.EnvolturaClave.EsquemaParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	perfil, claveRef, versionClave, material, huellaAAD, huellaEnvoltura, err :=
		solicitud.EnvolturaClave.DatosParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	envoltura, err := RestaurarEnvolturaClaveKMSBorrador(
		esquemaEnvoltura, perfil, claveRef, versionClave, material, huellaAAD, huellaEnvoltura,
	)
	if err != nil {
		t.Fatal(err)
	}
	esquemaSobre, err := solicitud.SobreCifrado.EsquemaParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	perfilSobre, nonce, cifrado, huellaAADSobre, huellaSobre, err :=
		solicitud.SobreCifrado.DatosParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	sobre, err := RestaurarSobreCifradoAEADBorrador(
		esquemaSobre, perfilSobre, nonce, cifrado, huellaAADSobre, huellaSobre,
	)
	if err != nil {
		t.Fatal(err)
	}
	restaurada := solicitud
	restaurada.EnvolturaClave = envoltura
	restaurada.SobreCifrado = sobre
	if err := restaurada.Validar(); err != nil {
		t.Fatalf("sobre durable restaurado inválido: %v", err)
	}
	if _, err := RestaurarEnvolturaClaveKMSBorrador(
		"esquema:ajeno", perfil, claveRef, versionClave, material, huellaAAD, huellaEnvoltura,
	); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("envoltura con esquema ajeno aceptada: %v", err)
	}
	if _, err := RestaurarSobreCifradoAEADBorrador(
		esquemaSobre, perfilSobre, nonce, cifrado, huellaAADSobre, huellaHexPrueba('0'),
	); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("sobre con huella almacenada ajena aceptado: %v", err)
	}
}

func TestAtestacionPersistidaSeRestauraSinClavePrivada(t *testing.T) {
	solicitud, _ := escenarioDescifradoBorradorPrueba(t)
	atestacion := solicitud.AtestacionKMS
	firmaBase64, err := atestacion.Firma.FirmaBase64URLParaPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	firma, err := RestaurarFirmaEvidenciaBorradorPersistida(
		atestacion.Firma.AlgoritmoFirma, atestacion.Firma.VerificadorRef,
		atestacion.Firma.HuellaClavePublicaSHA256, atestacion.Firma.HuellaPreimagenSHA256,
		firmaBase64,
	)
	if err != nil {
		t.Fatal(err)
	}
	restaurada, err := RestaurarAtestacionKMSBorrador(
		atestacion.Esquema, atestacion.AtestacionRef, atestacion.VersionAtestacion,
		atestacion.Estado, atestacion.Perfil, atestacion.ClaveMaestraRef,
		atestacion.VersionClave, atestacion.HuellaAAD, atestacion.HuellaEnvolturaSHA256,
		atestacion.HuellaSobreSHA256, atestacion.VerificadorRef, atestacion.Procedencia,
		firma, atestacion.EmitidaEn, atestacion.ValidaHasta,
	)
	if err != nil || restaurada != atestacion {
		t.Fatalf("atestación durable no restaurada exactamente: %v", err)
	}
	if _, err := RestaurarAtestacionKMSBorrador(
		"esquema:ajeno", atestacion.AtestacionRef, atestacion.VersionAtestacion,
		atestacion.Estado, atestacion.Perfil, atestacion.ClaveMaestraRef,
		atestacion.VersionClave, atestacion.HuellaAAD, atestacion.HuellaEnvolturaSHA256,
		atestacion.HuellaSobreSHA256, atestacion.VerificadorRef, atestacion.Procedencia,
		firma, atestacion.EmitidaEn, atestacion.ValidaHasta,
	); !errors.Is(err, ErrDescifradoBorradorInvalido) {
		t.Fatalf("atestación con esquema ajeno aceptada: %v", err)
	}
}
