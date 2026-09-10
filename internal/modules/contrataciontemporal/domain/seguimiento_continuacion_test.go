package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func fixtureContinuacionSeguimiento(t *testing.T) (DefinicionSeguimiento, DefinicionSeguimiento, Seguimiento, DatosTransicionSeguimiento) {
	t.Helper()
	base := definicionSeguimientoValida(t, false).Publicacion()
	borrador := borradorContinuacionDesdePublicacion(base)
	borrador.Version = 1
	borrador.Estados = []EstadoDefinidoSeguimiento{{Clave: "pendiente_incorporacion"}, {Clave: "vigente"}, {Clave: "cancelada", Final: true}}
	borrador.Motivos = []ClaveCatalogo{"necesidad_servicio", "motivo_historico"}
	borrador.Transiciones = nil
	for _, tr := range base.Transiciones {
		if tr.Clave == "confirmar_incorporacion" {
			borrador.Transiciones = append(borrador.Transiciones, tr)
		}
	}
	original, err := PublicarDefinicionSeguimiento(borrador)
	if err != nil {
		t.Fatal(err)
	}
	seguimiento := seguimientoIncorporado(t, original)
	datos := datosSeguimiento("acto_cierre_administrativo", TransicionCerrarAdministrativamenteSinCese, 2)
	datos.RegistradaEn = instanteSeguimientoBase.Add(9 * 24 * time.Hour)
	datos.EfectivoEn = datos.RegistradaEn
	datos.MotivoClave = "cierre_administrativo_ejercicio"
	datos.Documentos = []DocumentoSeguimiento{{TipoClave: "anotacion_administrativa", Referencia: referenciaSeguimientoPrueba("anotacion_previa")}}
	sucesora := publicarSucesoraContinuacion(t, original, datos.RegistradaEn)
	return original, sucesora, seguimiento, datos
}

func publicarSucesoraContinuacion(t *testing.T, original DefinicionSeguimiento, cierre time.Time) DefinicionSeguimiento {
	t.Helper()
	b := borradorContinuacionDesdePublicacion(original.Publicacion())
	b.Version++
	b.PublicadoEn = cierre.Add(-time.Hour)
	b.Vigencia = VigenciaSeguimiento{Desde: b.PublicadoEn, Hasta: cierre.AddDate(1, 0, 0)}
	b.Estados = append(b.Estados, EstadoDefinidoSeguimiento{Clave: EstadoCerradoAdministrativamenteSeguimiento, Final: true})
	b.Motivos = append(b.Motivos, "cierre_administrativo_ejercicio")
	b.Transiciones = append(b.Transiciones, TransicionDefinidaSeguimiento{
		Clave: TransicionCerrarAdministrativamenteSinCese, Origen: "vigente", Destino: EstadoCerradoAdministrativamenteSeguimiento,
		Clase: TransicionOrdinaria, MotivoObligatorio: true, MotivosPermitidos: []ClaveCatalogo{"cierre_administrativo_ejercicio"},
		Documentos:    []RequisitoDocumentoSeguimiento{{TipoClave: "anotacion_administrativa", Obligatorio: true}},
		EfectoPeriodo: EfectoPeriodoNinguno,
	})
	sucesora, err := PublicarDefinicionSeguimiento(b)
	if err != nil {
		t.Fatal(err)
	}
	return sucesora
}

func borradorContinuacionDesdePublicacion(p PublicacionDefinicionSeguimiento) BorradorDefinicionSeguimiento {
	return BorradorDefinicionSeguimiento{
		Referencia: p.Referencia, Version: p.Version, PublicadoEn: p.PublicadoEn, Vigencia: p.Vigencia,
		EstadoInicial: p.EstadoInicial, ProhibeCiclosSilenciosos: p.ProhibeCiclosSilenciosos,
		Estados: p.Estados, Motivos: p.Motivos, Transiciones: p.Transiciones,
	}
}

func cierreContinuacionPrueba(t *testing.T, original, sucesora DefinicionSeguimiento, previo Seguimiento, datos DatosTransicionSeguimiento) Seguimiento {
	t.Helper()
	continuacion, err := NuevaContinuacionSeguimiento(original, sucesora, previo, datos)
	if err != nil {
		t.Fatalf("preparar continuacion: %v", err)
	}
	resultado, err := AplicarCierreConContinuacion(previo, original, sucesora, continuacion, previo.Version(), datos)
	if err != nil {
		t.Fatalf("aplicar continuacion: %v", err)
	}
	return resultado
}

func TestSeguimientoContinuacionCierraSinCeseYConservaHistoriaV1(t *testing.T) {
	original, sucesora, previo, datos := fixtureContinuacionSeguimiento(t)
	publicacionAntes := original.Publicacion()
	estadoAntes := previo.Estado()
	canonAntes, err := SerializarEstadoSeguimientoCanonico(original, estadoAntes)
	if err != nil {
		t.Fatal(err)
	}
	jsonAntes, _ := json.Marshal(estadoAntes)
	if bytes.Contains(jsonAntes, []byte("continuacion")) {
		t.Fatal("V1 expone nuevo campo opcional")
	}
	if _, err := previo.Aplicar(original, previo.Version(), datos); !errors.Is(err, ErrTransicionInvalida) {
		t.Fatalf("V1 autorizo cierre inexistente: %v", err)
	}
	if _, err := previo.Aplicar(sucesora, previo.Version(), datos); err == nil {
		t.Fatal("la sucesora sustituyo fundacion")
	}

	cerrado := cierreContinuacionPrueba(t, original, sucesora, previo, datos)
	estado := cerrado.Estado()
	if cerrado.Version() != previo.Version()+1 || cerrado.EstadoActual() != EstadoCerradoAdministrativamenteSeguimiento || estado.Continuacion == nil {
		t.Fatal("cierre nominal incompleto")
	}
	if estado.HuellaRaizSHA256 != estadoAntes.HuellaRaizSHA256 || estado.Definicion != estadoAntes.Definicion ||
		estado.Referencia != estadoAntes.Referencia || estado.RelacionRef != estadoAntes.RelacionRef ||
		!reflect.DeepEqual(estado.Actuaciones[:len(estado.Actuaciones)-1], estadoAntes.Actuaciones) ||
		!reflect.DeepEqual(estado.PeriodosResultantes, estadoAntes.PeriodosResultantes) ||
		estado.PeriodoPrevisto != estadoAntes.PeriodoPrevisto || !reflect.DeepEqual(estado.CeseEfectivo, estadoAntes.CeseEfectivo) {
		t.Fatal("cierre reescribio raiz, prefijo o periodos")
	}
	ultima := estado.Actuaciones[len(estado.Actuaciones)-1]
	if ultima.Definicion != sucesora.Referencia() || ultima.HuellaAnteriorSHA256 != estadoAntes.Actuaciones[len(estadoAntes.Actuaciones)-1].HuellaActuacionSHA256 {
		t.Fatal("nueva actuacion no liga sucesora y prefijo")
	}
	if !reflect.DeepEqual(original.Publicacion(), publicacionAntes) || !reflect.DeepEqual(previo.Estado(), estadoAntes) {
		t.Fatal("se mutaron valores de entrada")
	}
	canonDespues, err := SerializarEstadoSeguimientoCanonico(original, previo.Estado())
	if err != nil || !bytes.Equal(canonAntes, canonDespues) {
		t.Fatal("canon V1 cambiado")
	}
}

func TestSeguimientoContinuacionSeRecuperaConCanonDistintoYV1LaRechaza(t *testing.T) {
	original, sucesora, previo, datos := fixtureContinuacionSeguimiento(t)
	cerrado := cierreContinuacionPrueba(t, original, sucesora, previo, datos)
	canon, err := SerializarEstadoSeguimientoConContinuacionCanonico(original, sucesora, cerrado.Estado())
	if err != nil || !bytes.Contains(canon, []byte(dominioEstadoSeguimientoContinuadoV1)) {
		t.Fatalf("canon continuado: %v", err)
	}
	contenido, err := json.Marshal(cerrado.Estado())
	if err != nil {
		t.Fatal(err)
	}
	var recuperado EstadoPersistidoSeguimiento
	if err := json.Unmarshal(contenido, &recuperado); err != nil {
		t.Fatal(err)
	}
	restaurado, err := RehidratarSeguimientoConContinuacion(original, sucesora, recuperado)
	if err != nil || !reflect.DeepEqual(restaurado.Estado(), cerrado.Estado()) {
		t.Fatalf("recuperacion no identica: %v", err)
	}
	canon2, err := SerializarEstadoSeguimientoConContinuacionCanonico(original, sucesora, restaurado.Estado())
	if err != nil || !bytes.Equal(canon, canon2) {
		t.Fatal("canon tras recuperacion cambio")
	}
	for _, d := range []DefinicionSeguimiento{original, sucesora} {
		if _, err := RehidratarSeguimiento(d, recuperado); err == nil {
			t.Fatal("rehidratador V1 acepto continuacion")
		}
		if _, err := SerializarEstadoSeguimientoCanonico(d, recuperado); err == nil {
			t.Fatal("codec V1 acepto continuacion")
		}
		if err := cerrado.Validar(d); err == nil {
			t.Fatal("Validar V1 acepto continuacion")
		}
		if _, err := cerrado.Aplicar(d, cerrado.Version(), datos); err == nil {
			t.Fatal("Aplicar V1 acepto continuacion")
		}
	}
	if _, err := NuevaContinuacionSeguimiento(original, sucesora, cerrado, datos); err == nil {
		t.Fatal("se admitio segunda adopcion")
	}
	copia := cerrado.Estado()
	copia.Continuacion.ReciboRef = referenciaSeguimientoPrueba("recibo_adulterado")
	copia.Actuaciones[0].Documentos[0].Referencia = referenciaSeguimientoPrueba("documento_adulterado")
	if reflect.DeepEqual(copia, cerrado.Estado()) || cerrado.Estado().Continuacion.ReciboRef != datos.ReciboRef {
		t.Fatal("copia expuso memoria mutable")
	}
}

func TestSeguimientoContinuacionRechazaAdulteracionesAunqueSeRecalculeHuellaFinal(t *testing.T) {
	original, sucesora, previo, datos := fixtureContinuacionSeguimiento(t)
	cerrado := cierreContinuacionPrueba(t, original, sucesora, previo, datos)
	casos := map[string]func(*EstadoPersistidoSeguimiento){
		"raiz": func(e *EstadoPersistidoSeguimiento) {
			e.HuellaRaizSHA256 = referenciaSeguimientoPrueba("otra_raiz")[4:]
		},
		"referencia": func(e *EstadoPersistidoSeguimiento) { e.Referencia = referenciaSeguimientoPrueba("otro_seguimiento") },
		"relacion":   func(e *EstadoPersistidoSeguimiento) { e.RelacionRef = referenciaSeguimientoPrueba("otra_relacion") },
		"fundacion":  func(e *EstadoPersistidoSeguimiento) { e.Definicion = sucesora.Referencia() },
		"version":    func(e *EstadoPersistidoSeguimiento) { e.Version++ },
		"estado":     func(e *EstadoPersistidoSeguimiento) { e.EstadoActual = "vigente" },
		"fecha":      func(e *EstadoPersistidoSeguimiento) { e.ActualizadoEn = e.ActualizadoEn.Add(time.Microsecond) },
		"fecha_no_utc": func(e *EstadoPersistidoSeguimiento) {
			e.ActualizadoEn = e.ActualizadoEn.In(time.FixedZone("otro", 3600))
		},
		"periodo": func(e *EstadoPersistidoSeguimiento) {
			e.PeriodosResultantes[0].Intervalo.Hasta = e.PeriodosResultantes[0].Intervalo.Hasta.Add(time.Hour)
		},
		"cese": func(e *EstadoPersistidoSeguimiento) {
			e.CeseEfectivo = &CeseEfectivoSeguimiento{EfectivoEn: datos.RegistradaEn, ActuacionRef: datos.ActuacionRef}
		},
		"sin_adopcion":      func(e *EstadoPersistidoSeguimiento) { e.Continuacion = nil },
		"version_anterior":  func(e *EstadoPersistidoSeguimiento) { e.Continuacion.VersionAnterior++ },
		"primera_secuencia": func(e *EstadoPersistidoSeguimiento) { e.Continuacion.PrimeraSecuencia++ },
		"hash_anterior": func(e *EstadoPersistidoSeguimiento) {
			e.Continuacion.HuellaEstadoAnteriorSHA256 = referenciaSeguimientoPrueba("hash_falso")[4:]
		},
		"publicacion_adoptada": func(e *EstadoPersistidoSeguimiento) { e.Continuacion.DefinicionSucesora = original.Referencia() },
		"recibo_adopcion": func(e *EstadoPersistidoSeguimiento) {
			e.Continuacion.ReciboRef = referenciaSeguimientoPrueba("recibo_falso")
		},
		"historia": func(e *EstadoPersistidoSeguimiento) {
			e.Actuaciones[0].ActorRef = referenciaSeguimientoPrueba("actor_falso")
		},
		"efecto_con_huella_nueva": func(e *EstadoPersistidoSeguimiento) {
			a := &e.Actuaciones[len(e.Actuaciones)-1]
			a.EstadoDestino = "vigente"
			a.HuellaActuacionSHA256, _ = calcularHuellaActuacionSeguimiento(*a)
		},
		"actuacion_definicion": func(e *EstadoPersistidoSeguimiento) {
			a := &e.Actuaciones[len(e.Actuaciones)-1]
			a.Definicion = original.Referencia()
			a.HuellaActuacionSHA256, _ = calcularHuellaActuacionSeguimiento(*a)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			estado := cerrado.Estado()
			alterar(&estado)
			if _, err := RehidratarSeguimientoConContinuacion(original, sucesora, estado); !errors.Is(err, ErrContinuacionSeguimientoInvalida) {
				t.Fatalf("adulteracion admitida: %v", err)
			}
			if _, err := SerializarEstadoSeguimientoConContinuacionCanonico(original, sucesora, estado); err == nil {
				t.Fatal("codec sello estado adulterado")
			}
		})
	}
}

func TestSeguimientoContinuacionExigeSucesoraNominalSinReinterpretarPublicacion(t *testing.T) {
	original, sucesora, previo, datos := fixtureContinuacionSeguimiento(t)
	casos := map[string]func(*BorradorDefinicionSeguimiento){
		"otra_referencia": func(b *BorradorDefinicionSeguimiento) { b.Referencia = referenciaSeguimientoPrueba("otra_publicacion") },
		"salto_version":   func(b *BorradorDefinicionSeguimiento) { b.Version++ },
		"misma_version":   func(b *BorradorDefinicionSeguimiento) { b.Version = original.Referencia().Version },
		"publicacion_futura": func(b *BorradorDefinicionSeguimiento) {
			b.PublicadoEn = datos.RegistradaEn.Add(time.Hour)
			b.Vigencia.Desde = b.PublicadoEn
		},
		"sucesora_expirada":        func(b *BorradorDefinicionSeguimiento) { b.Vigencia.Hasta = datos.RegistradaEn },
		"nueva_vigencia_futura":    func(b *BorradorDefinicionSeguimiento) { b.Vigencia.Desde = datos.RegistradaEn.Add(time.Hour) },
		"publicacion_no_posterior": func(b *BorradorDefinicionSeguimiento) { b.PublicadoEn = original.Publicacion().PublicadoEn },
		"vigencia_retroactiva":     func(b *BorradorDefinicionSeguimiento) { b.Vigencia.Desde = b.PublicadoEn.Add(-time.Hour) },
		"estado_inicial":           func(b *BorradorDefinicionSeguimiento) { b.EstadoInicial = "vigente" },
		"politica_ciclos":          func(b *BorradorDefinicionSeguimiento) { b.ProhibeCiclosSilenciosos = false },
		"cambio_historico": func(b *BorradorDefinicionSeguimiento) {
			for i := range b.Transiciones {
				if b.Transiciones[i].Clave == "confirmar_incorporacion" {
					b.Transiciones[i].Documentos[0].Obligatorio = false
				}
			}
		},
		"motivo_historico_retirado": func(b *BorradorDefinicionSeguimiento) {
			m := []ClaveCatalogo{}
			for _, v := range b.Motivos {
				if v != "motivo_historico" {
					m = append(m, v)
				}
			}
			b.Motivos = m
		},
		"estado_adicional": func(b *BorradorDefinicionSeguimiento) {
			b.Estados = append(b.Estados, EstadoDefinidoSeguimiento{Clave: "otro_estado"})
		},
		"cierre_crea_cese": func(b *BorradorDefinicionSeguimiento) {
			for i := range b.Transiciones {
				if b.Transiciones[i].Clave == TransicionCerrarAdministrativamenteSinCese {
					b.Transiciones[i].EfectoPeriodo = EfectoPeriodoCerrar
				}
			}
		},
		"cierre_no_final": func(b *BorradorDefinicionSeguimiento) {
			for i := range b.Estados {
				if b.Estados[i].Clave == EstadoCerradoAdministrativamenteSeguimiento {
					b.Estados[i].Final = false
				}
			}
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			b := borradorContinuacionDesdePublicacion(sucesora.Publicacion())
			alterar(&b)
			d, err := PublicarDefinicionSeguimiento(b)
			// La propia publicacion puede rechazar vigencias o catalogos invalidos.
			if err != nil {
				return
			}
			if _, err := NuevaContinuacionSeguimiento(original, d, previo, datos); err == nil {
				t.Fatal("sucesora no nominal aceptada")
			}
		})
	}
}

func TestSeguimientoContinuacionLigaSolicitudVersionYReciboExactos(t *testing.T) {
	original, sucesora, previo, datos := fixtureContinuacionSeguimiento(t)
	continuacion, err := NuevaContinuacionSeguimiento(original, sucesora, previo, datos)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AplicarCierreConContinuacion(previo, original, sucesora, continuacion, previo.Version()+1, datos); !errors.Is(err, ErrVersionEnConflicto) {
		t.Fatalf("version: %v", err)
	}
	casos := map[string]func(*DatosTransicionSeguimiento){
		"actor":  func(d *DatosTransicionSeguimiento) { d.ActorRef = referenciaSeguimientoPrueba("otro_actor") },
		"unidad": func(d *DatosTransicionSeguimiento) { d.UnidadRef = referenciaSeguimientoPrueba("otra_unidad") },
		"correlacion": func(d *DatosTransicionSeguimiento) {
			d.CorrelacionRef = referenciaSeguimientoPrueba("otra_correlacion")
		},
		"recibo":        func(d *DatosTransicionSeguimiento) { d.ReciboRef = referenciaSeguimientoPrueba("otro_recibo") },
		"actuacion":     func(d *DatosTransicionSeguimiento) { d.ActuacionRef = referenciaSeguimientoPrueba("otra_actuacion") },
		"motivo":        func(d *DatosTransicionSeguimiento) { d.MotivoClave = "necesidad_servicio" },
		"transicion":    func(d *DatosTransicionSeguimiento) { d.TransicionClave = "confirmar_incorporacion" },
		"sin_documento": func(d *DatosTransicionSeguimiento) { d.Documentos = nil },
		"documento": func(d *DatosTransicionSeguimiento) {
			d.Documentos[0].Referencia = referenciaSeguimientoPrueba("otro_documento")
		},
		"efecto_futuro": func(d *DatosTransicionSeguimiento) { d.EfectivoEn = d.EfectivoEn.Add(time.Hour) },
		"periodo":       func(d *DatosTransicionSeguimiento) { d.Periodo = punteroIntervalo(previo.Estado().PeriodoPrevisto) },
		"fecha": func(d *DatosTransicionSeguimiento) {
			d.RegistradaEn = d.RegistradaEn.Add(time.Microsecond)
			d.EfectivoEn = d.RegistradaEn
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			d := datos.clonar()
			alterar(&d)
			if err := ValidarContinuacionSeguimiento(original, sucesora, previo, continuacion, d); err == nil {
				t.Fatal("descriptor acepto otro material")
			}
		})
	}
	for _, caso := range []struct {
		nombre  string
		alterar func(*DatosTransicionSeguimiento)
	}{
		{"actuacion_existente", func(d *DatosTransicionSeguimiento) { d.ActuacionRef = previo.Actuaciones()[0].ActuacionRef }},
		{"recibo_existente", func(d *DatosTransicionSeguimiento) { d.ReciboRef = previo.Actuaciones()[0].ReciboRef }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			d := datos.clonar()
			caso.alterar(&d)
			if _, err := NuevaContinuacionSeguimiento(original, sucesora, previo, d); !errors.Is(err, ErrActuacionSeguimientoEnConflicto) {
				t.Fatalf("identidad repetida: %v", err)
			}
		})
	}
	pendiente := seguimientoNuevoValido(t, original)
	if _, err := NuevaContinuacionSeguimiento(original, sucesora, pendiente, datos); err == nil {
		t.Fatal("se cerro seguimiento sin incorporacion")
	}
}

func TestSeguimientoContinuacionPermitePublicacionFundacionalExpiradaSinReescribirla(t *testing.T) {
	original, _, _, datos := fixtureContinuacionSeguimiento(t)
	b := borradorContinuacionDesdePublicacion(original.Publicacion())
	b.Vigencia.Hasta = instanteSeguimientoBase.Add(2 * 24 * time.Hour)
	original, err := PublicarDefinicionSeguimiento(b)
	if err != nil {
		t.Fatal(err)
	}
	previo := seguimientoIncorporado(t, original)
	sucesora := publicarSucesoraContinuacion(t, original, datos.RegistradaEn)
	if original.VigenteEn(datos.RegistradaEn) {
		t.Fatal("fixture no expira")
	}
	cerrado := cierreContinuacionPrueba(t, original, sucesora, previo, datos)
	if _, err := RehidratarSeguimientoConContinuacion(original, sucesora, cerrado.Estado()); err != nil {
		t.Fatal(err)
	}
	if original.Publicacion().Vigencia.Hasta != b.Vigencia.Hasta {
		t.Fatal("se amplio vigencia original")
	}
}
