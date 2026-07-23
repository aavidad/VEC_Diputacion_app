package ports

import (
	"context"
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"
)

type presentadorAutoridadConfiguradoPrueba struct {
	datos        DatosCredencialAutoridadFuenteAnalisis
	clavePrueba  ed25519.PrivateKey
	claveRaiz    ed25519.PrivateKey
	alterar      func(*CredencialAutoridadFuenteAnalisis)
	reusarPrueba []byte
}

func nuevoPresentadorAutoridadConfiguradoPrueba(
	rol RolAutoridadFuenteAnalisis,
	autoridadRef string,
	backendRef string,
) presentadorAutoridadConfiguradoPrueba {
	datos := datosCredencialAutoridadPrueba(rol, autoridadRef, backendRef)
	return presentadorAutoridadConfiguradoPrueba{
		datos:       datos,
		clavePrueba: claveEd25519Prueba(string(rol) + ":" + autoridadRef),
		claveRaiz:   claveEd25519Prueba("raiz-institucional"),
	}
}

func (p presentadorAutoridadConfiguradoPrueba) PresentarAutoridadFuenteAnalisis(
	_ context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	documento, err := canonCredencialAutoridadFuenteAnalisis(p.datos)
	if err != nil {
		return PresentacionAutoridadFuenteAnalisis{}, err
	}
	credencial, err := NuevaCredencialAutoridadFuenteAnalisis(
		p.datos,
		ed25519.Sign(p.claveRaiz, documento),
	)
	if err != nil {
		return PresentacionAutoridadFuenteAnalisis{}, err
	}
	if p.alterar != nil {
		p.alterar(&credencial)
	}
	prueba := append([]byte(nil), p.reusarPrueba...)
	if len(prueba) == 0 {
		material, errMaterial := desafio.Bytes()
		if errMaterial != nil {
			return PresentacionAutoridadFuenteAnalisis{}, errMaterial
		}
		prueba = ed25519.Sign(p.clavePrueba, material)
	}
	return NuevaPresentacionAutoridadFuenteAnalisis(credencial, prueba)
}

type calculadorAutoridadConfiguradaPrueba struct {
	presentador presentadorAutoridadConfiguradoPrueba
	invocado    *bool
}

func (c calculadorAutoridadConfiguradaPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return c.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (c calculadorAutoridadConfiguradaPrueba) CalcularCoste(
	context.Context,
	SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	*c.invocado = true
	return ResultadoCalculoCoste{}, nil
}

type verificadorAutoridadConfiguradaPrueba struct {
	presentador presentadorAutoridadConfiguradoPrueba
}

func (v verificadorAutoridadConfiguradaPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return v.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (verificadorAutoridadConfiguradaPrueba) VerificarRespuestaFuenteAnalisis(
	context.Context,
	SolicitudVerificarRespuestaFuenteAnalisis,
) (ConfirmacionRespuestaFuenteAnalisis, error) {
	return ConfirmacionRespuestaFuenteAnalisis{}, nil
}

type publicadorQueUsaIdentidadFuentePrueba struct {
	presentador presentadorAutoridadConfiguradoPrueba
}

func (p publicadorQueUsaIdentidadFuentePrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio DesafioAutoridadFuenteAnalisis,
) (PresentacionAutoridadFuenteAnalisis, error) {
	return p.presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (publicadorQueUsaIdentidadFuentePrueba) VerificarPublicacionMotivoFuenteAnalisis(
	context.Context,
	SolicitudVerificarPublicacionMotivoFuenteAnalisis,
) (ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	return ConfirmacionPublicacionMotivoFuenteAnalisis{}, nil
}

func TestWrappersDistintosDelMismoBackendNoSeparanTCB(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	invocado := false
	calculador := calculadorAutoridadConfiguradaPrueba{
		presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
			RolCalculadorCoste,
			"tabla_retributiva_2026_v3",
			"backend_compartido_analisis_012345",
		),
		invocado: &invocado,
	}
	verificador := verificadorAutoridadConfiguradaPrueba{
		presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
			RolVerificadorRespuesta,
			"verificador_alias_tcb_0123456789",
			"backend_compartido_analisis_012345",
		),
	}
	_, err := calcularCosteConFuenteOrquestadoPrueba(
		context.Background(),
		calculador,
		verificador,
		consumidorRespuestaPrueba(inicio),
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(inicio),
		solicitudCalcularCostePrueba(t, inicio),
	)
	if !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) || invocado {
		t.Fatalf("wrappers del mismo backend aceptados: %v", err)
	}
}

func TestFuenteNoPuedeAutopublicarSuCatalogo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	malicioso := publicadorQueUsaIdentidadFuentePrueba{
		presentador: nuevoPresentadorAutoridadConfiguradoPrueba(
			RolFuentePresupuestaria,
			"fuente_presupuesto_0123456789",
			"backend_presupuesto_0123456789",
		),
	}
	_, err := validarRCConFuenteOrquestadaPrueba(
		context.Background(),
		fuentePresupuestariaDoble(func(
			context.Context,
			SolicitudValidarRC,
		) (ResultadoValidacionRC, error) {
			t.Fatal("la fuente no debe invocarse sin separación completa")
			return ResultadoValidacionRC{}, nil
		}),
		verificadorRespuestaHMACPrueba(inicio),
		malicioso,
		consumidorRespuestaPrueba(inicio),
		confianzaAutoridadesPrueba(t),
		relojFijoFuenteAnalisis(inicio),
		solicitudValidarRCPrueba(t, inicio),
	)
	if !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("la fuente autopublicó el catálogo: %v", err)
	}
}

func TestSeparacionRechazaAliasesIdentidadesYClavesRepetidas(t *testing.T) {
	claveA := claveEd25519Prueba("identidad-a").Public().(ed25519.PublicKey)
	claveB := claveEd25519Prueba("identidad-b").Public().(ed25519.PublicKey)
	base := identidadAutoridadFuenteAnalisis{
		autoridadRef: "autoridad_fuente_0123456789",
		backendRef:   "backend_canonico_0123456789",
		clavePrueba:  claveA,
		rol:          RolFuentePresupuestaria,
	}
	casos := []struct {
		nombre string
		otra   identidadAutoridadFuenteAnalisis
	}{
		{
			nombre: "alias mismo backend",
			otra: identidadAutoridadFuenteAnalisis{
				autoridadRef: "alias_verificador_0123456789",
				backendRef:   base.backendRef,
				clavePrueba:  claveB,
				rol:          RolVerificadorRespuesta,
			},
		},
		{
			nombre: "identidad repetida",
			otra: identidadAutoridadFuenteAnalisis{
				autoridadRef: base.autoridadRef,
				backendRef:   "backend_verificador_0123456789",
				clavePrueba:  claveB,
				rol:          RolVerificadorRespuesta,
			},
		},
		{
			nombre: "clave repetida",
			otra: identidadAutoridadFuenteAnalisis{
				autoridadRef: "verificador_tcb_otro_012345",
				backendRef:   "backend_verificador_otro_012345",
				clavePrueba:  claveA,
				rol:          RolVerificadorRespuesta,
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if autoridadesFuenteAnalisisSeparadasPrueba(base, caso.otra) {
				t.Fatal("se aceptó una separación de autoridad aparente")
			}
		})
	}
}

func TestCredencialManipuladaOAutoemitidaFallaCerrado(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	material := []byte("peticion-canonica-prueba")
	confianza := confianzaAutoridadesPrueba(t)
	casos := []struct {
		nombre      string
		presentador presentadorAutoridadConfiguradoPrueba
	}{
		{
			nombre: "backend manipulado tras firma",
			presentador: func() presentadorAutoridadConfiguradoPrueba {
				p := nuevoPresentadorAutoridadConfiguradoPrueba(
					RolCalculadorCoste,
					"tabla_retributiva_2026_v3",
					"backend_calculo_coste_0123456789",
				)
				p.alterar = func(c *CredencialAutoridadFuenteAnalisis) {
					c.datos.BackendRef = "backend_atacante_0123456789"
				}
				return p
			}(),
		},
		{
			nombre: "raiz autoemitida",
			presentador: func() presentadorAutoridadConfiguradoPrueba {
				p := nuevoPresentadorAutoridadConfiguradoPrueba(
					RolCalculadorCoste,
					"tabla_retributiva_2026_v3",
					"backend_calculo_coste_0123456789",
				)
				p.claveRaiz = claveEd25519Prueba("raiz-atacante")
				return p
			}(),
		},
		{
			nombre: "organizacion firmada ajena",
			presentador: func() presentadorAutoridadConfiguradoPrueba {
				p := nuevoPresentadorAutoridadConfiguradoPrueba(
					RolCalculadorCoste,
					"tabla_retributiva_2026_v3",
					"backend_calculo_coste_0123456789",
				)
				p.datos.OrganizacionRef = "organizacion_ajena_0123456789"
				return p
			}(),
		},
		{
			nombre: "audiencia firmada ajena",
			presentador: func() presentadorAutoridadConfiguradoPrueba {
				p := nuevoPresentadorAutoridadConfiguradoPrueba(
					RolCalculadorCoste,
					"tabla_retributiva_2026_v3",
					"backend_calculo_coste_0123456789",
				)
				p.datos.Audiencia = "audiencia_ajena_0123456789"
				return p
			}(),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := presentarYVerificarAutoridadFuenteAnalisisPrueba(
				context.Background(),
				caso.presentador,
				confianza,
				material,
				RolCalculadorCoste,
				inicio,
			)
			if !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
				t.Fatalf("credencial no confiable aceptada: %v", err)
			}
		})
	}
}

func TestPruebaDePosesionNoSePuedeCopiarEntreDesafios(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	confianza := confianzaAutoridadesPrueba(t)
	presentador := nuevoPresentadorAutoridadConfiguradoPrueba(
		RolCalculadorCoste,
		"tabla_retributiva_2026_v3",
		"backend_calculo_coste_0123456789",
	)
	desafioPrimero, err := nuevoDesafioAutoridadFuenteAnalisis(
		[]byte("peticion-canonica"),
		organizacionAutoridadPrueba,
		audienciaAutoridadPrueba,
		RolCalculadorCoste,
	)
	if err != nil {
		t.Fatal(err)
	}
	presentacion, err := presentador.PresentarAutoridadFuenteAnalisis(
		context.Background(),
		desafioPrimero,
	)
	if err != nil {
		t.Fatal(err)
	}
	desafioSegundo, err := nuevoDesafioAutoridadFuenteAnalisis(
		[]byte("peticion-canonica"),
		organizacionAutoridadPrueba,
		audienciaAutoridadPrueba,
		RolCalculadorCoste,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := confianza.verificarPresentacion(
		presentacion,
		desafioSegundo,
		RolCalculadorCoste,
		inicio,
	); err == nil {
		t.Fatal("una prueba copiada superó un nonce distinto")
	}
}

func TestRotacionYRevocacionDeAutoridades(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	raizAntigua := claveEd25519Prueba("raiz-antigua")
	raizNueva := claveEd25519Prueba("raiz-nueva")
	crearConfianza := func(
		estadoAntigua EstadoRaizAutoridadFuenteAnalisis,
		revocaciones []RevocacionAutoridadFuenteAnalisis,
	) ConfianzaAutoridadesFuenteAnalisis {
		t.Helper()
		confianza, err := NuevaConfianzaAutoridadesFuenteAnalisis(
			organizacionAutoridadPrueba,
			audienciaAutoridadPrueba,
			[]RaizConfianzaAutoridadFuenteAnalisis{
				{
					ClaveID:                "raiz_antigua_0123456789",
					ClavePublicaEd25519:    raizAntigua.Public().(ed25519.PublicKey),
					Estado:                 estadoAntigua,
					ValidaDesde:            inicio.AddDate(-1, 0, 0),
					ValidaHasta:            inicio.AddDate(1, 0, 0),
					UltimaEmisionPermitida: inicio,
				},
				{
					ClaveID:                "raiz_nueva_0123456789",
					ClavePublicaEd25519:    raizNueva.Public().(ed25519.PublicKey),
					Estado:                 RaizAutoridadActiva,
					ValidaDesde:            inicio.AddDate(-1, 0, 0),
					ValidaHasta:            inicio.AddDate(1, 0, 0),
					UltimaEmisionPermitida: inicio.AddDate(1, 0, 0),
				},
			},
			revocaciones,
		)
		if err != nil {
			t.Fatal(err)
		}
		return confianza
	}
	presentador := nuevoPresentadorAutoridadConfiguradoPrueba(
		RolCalculadorCoste,
		"tabla_retributiva_2026_v3",
		"backend_calculo_coste_0123456789",
	)
	presentador.datos.RaizClaveID = "raiz_antigua_0123456789"
	presentador.claveRaiz = raizAntigua
	if _, err := presentarYVerificarAutoridadFuenteAnalisisPrueba(
		context.Background(),
		presentador,
		crearConfianza(RaizAutoridadRetenida, nil),
		[]byte("peticion-canonica"),
		RolCalculadorCoste,
		inicio,
	); err != nil {
		t.Fatalf("raíz retenida durante rotación rechazada: %v", err)
	}
	revocacion := RevocacionAutoridadFuenteAnalisis{
		AutoridadRef: presentador.datos.AutoridadRef,
		Serie:        presentador.datos.Serie,
		RevocadaEn:   inicio,
	}
	if _, err := presentarYVerificarAutoridadFuenteAnalisisPrueba(
		context.Background(),
		presentador,
		crearConfianza(RaizAutoridadRetenida, []RevocacionAutoridadFuenteAnalisis{
			revocacion,
		}),
		[]byte("peticion-canonica"),
		RolCalculadorCoste,
		inicio,
	); !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("credencial revocada aceptada: %v", err)
	}
	if _, err := presentarYVerificarAutoridadFuenteAnalisisPrueba(
		context.Background(),
		presentador,
		crearConfianza(RaizAutoridadRevocada, nil),
		[]byte("peticion-canonica"),
		RolCalculadorCoste,
		inicio,
	); !errors.Is(err, ErrResultadoFuenteAnalisisNoConfiable) {
		t.Fatalf("raíz revocada aceptada: %v", err)
	}
}

func TestLimiteEnteroSeguroVersionExpedienteYCatalogo(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	preparacion := preparacionCalcularCostePrueba()
	preparacion.VersionExpediente = maximoEnteroSeguroFuenteAnalisis
	if _, err := nuevaSolicitudCalcularCosteOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	); err != nil {
		t.Fatalf("2^53-1 rechazado para expediente: %v", err)
	}
	preparacion.VersionExpediente++
	if _, err := nuevaSolicitudCalcularCosteOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	); err == nil {
		t.Fatal("2^53 aceptado para expediente")
	}
	if _, err := NuevoMotivoFuenteAnalisis(
		"catalogo_motivos_rc_0123456789",
		maximoEnteroSeguroFuenteAnalisis,
		strings.Repeat("a", 64),
		"motivo_publicado",
		"contratacion_temporal.rc.motivo_publicado",
		nil,
	); err != nil {
		t.Fatalf("2^53-1 rechazazado para catálogo: %v", err)
	}
	if _, err := NuevoMotivoFuenteAnalisis(
		"catalogo_motivos_rc_0123456789",
		maximoEnteroSeguroFuenteAnalisis+1,
		strings.Repeat("a", 64),
		"motivo_publicado",
		"contratacion_temporal.rc.motivo_publicado",
		nil,
	); err == nil {
		t.Fatal("2^53 aceptado para catálogo")
	}
}
