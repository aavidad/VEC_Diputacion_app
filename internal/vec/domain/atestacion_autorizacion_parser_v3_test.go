package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestParsearMensajeAtestacionAutorizacionV3ProyectaCompromisosExactos(
	t *testing.T,
) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil {
		t.Fatal(err)
	}
	proyeccion, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(mensaje)
	if err != nil {
		t.Fatalf("parsear VEC-AD-3: %v", err)
	}
	cabeceraObtenida, errCabecera := proyeccion.Cabecera()
	referencia, errReferencia := proyeccion.DecisionRef()
	huellaDecision, errDecision := proyeccion.HuellaDecisionSHA256()
	huellaMotivo, errMotivo := proyeccion.HuellaMotivoSHA256()
	referenciaContexto, errReferenciaContexto := proyeccion.ReferenciaContextoActor()
	huellaContexto, errContexto := proyeccion.HuellaContextoActorSHA256()
	decisionCanonica, _ := RepresentacionCanonicaDecisionAutorizacionV3(decision)
	sumaDecision := sha256SumAtestacionV3Prueba(decisionCanonica)
	sumaMotivo, _ := HuellaSHA256MotivoAutorizacionV2(motivo)
	if errCabecera != nil || errReferencia != nil || errDecision != nil ||
		errMotivo != nil || errReferenciaContexto != nil || errContexto != nil ||
		cabeceraObtenida != cabecera ||
		referencia != decision.datos.decisionRef ||
		huellaDecision != sumaDecision ||
		huellaMotivo != sumaMotivo ||
		referenciaContexto != contexto.RegistroContextoRef ||
		huellaContexto != contexto.HuellaSHA256 {
		t.Fatalf("proyección incompleta o cruzada: %#v", proyeccion)
	}
}

func TestParsearMensajeAtestacionAutorizacionV3RechazaTruncadoYJSONNoCanonico(
	t *testing.T,
) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, longitud := range []int{
		0, 1, len(mensaje) / 2, len(mensaje) - 9, len(mensaje) - 1,
	} {
		if _, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
			mensaje[:longitud],
		); !errors.Is(err, ErrParseoAtestacionAutorizacionV3Invalido) {
			t.Fatalf("truncado %d aceptado: %v", longitud, err)
		}
	}
	concatenado := append(append([]byte(nil), mensaje...), 0)
	if _, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
		concatenado,
	); !errors.Is(err, ErrParseoAtestacionAutorizacionV3Invalido) {
		t.Fatalf("mensaje concatenado aceptado: %v", err)
	}

	desconocido := reescribirDecisionAtestacionV3Prueba(
		t,
		mensaje,
		func(decision []byte) []byte {
			return append([]byte(`{"campo_desconocido":1,`), decision[1:]...)
		},
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
		desconocido,
	); !errors.Is(err, ErrParseoAtestacionAutorizacionV3Invalido) {
		t.Fatalf("campo desconocido aceptado: %v", err)
	}

	duplicado := reescribirDecisionAtestacionV3Prueba(
		t,
		mensaje,
		func(decision []byte) []byte {
			parte := []byte(`,"decision_ref":"dec_otra234567890abcdef0123456789ab"}`)
			return append(append([]byte(nil), decision[:len(decision)-1]...), parte...)
		},
	)
	if _, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
		duplicado,
	); !errors.Is(err, ErrParseoAtestacionAutorizacionV3Invalido) {
		t.Fatalf("clave duplicada aceptada: %v", err)
	}
}

func TestParsearMensajeAtestacionAutorizacionV3RechazaMutacionesSemanticasRecomprometidas(
	t *testing.T,
) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	mensaje, err := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := []struct {
		nombre string
		mutar  func(*decisionAutorizacionCanonicaV3)
	}{
		{"accion vacia", func(d *decisionAutorizacionCanonicaV3) { d.Accion = "" }},
		{"recurso vacio", func(d *decisionAutorizacionCanonicaV3) { d.RecursoRef = "" }},
		{"modulo vacio", func(d *decisionAutorizacionCanonicaV3) { d.ModuloID = "" }},
		{"tipo vacio", func(d *decisionAutorizacionCanonicaV3) { d.TipoRecurso = "" }},
		{"finalidad vacia", func(d *decisionAutorizacionCanonicaV3) { d.Finalidad = "" }},
		{"correlacion vacia", func(d *decisionAutorizacionCanonicaV3) { d.CorrelacionRef = "" }},
		{"esquema solicitud", func(d *decisionAutorizacionCanonicaV3) {
			d.EsquemaHuellaSolicitud = "esquema-ajeno"
		}},
		{"huella solicitud nula", func(d *decisionAutorizacionCanonicaV3) {
			d.SolicitudHuellaSHA256 = strings.Repeat("0", 64)
		}},
		{"esquema motivo", func(d *decisionAutorizacionCanonicaV3) {
			d.EsquemaHuellaMotivo = "esquema-ajeno"
		}},
		{"principal cruzado", func(d *decisionAutorizacionCanonicaV3) {
			d.PrincipalID = "per_otra234567890abcdefghijklmn"
		}},
		{"perfil cruzado", func(d *decisionAutorizacionCanonicaV3) {
			d.PerfilActivoRef = "prf_otra234567890abcdefghijklmn"
		}},
		{"control rol cruzado", func(d *decisionAutorizacionCanonicaV3) {
			d.ControlVigenciaVersionRolRef = "rol_otra234567890abcdefghijklmn"
		}},
		{"revision catalogo cero", func(d *decisionAutorizacionCanonicaV3) {
			d.RevisionCatalogoPoliticas = 0
		}},
		{"huella catalogo cruzada", func(d *decisionAutorizacionCanonicaV3) {
			d.CatalogoPoliticasHuellaSHA256 = strings.Repeat("a", 64)
		}},
		{"garantia invalida", func(d *decisionAutorizacionCanonicaV3) {
			d.GarantiaMinima = AuthAssurance("no-admitida")
		}},
		{"ventana invertida", func(d *decisionAutorizacionCanonicaV3) {
			d.ValidaHasta = d.EmitidaEn
		}},
		{"vinculo sin autenticacion", func(d *decisionAutorizacionCanonicaV3) {
			d.VinculoAutenticacionActor.AutenticacionRef = ""
		}},
		{"vinculo contexto ajeno", func(d *decisionAutorizacionCanonicaV3) {
			d.VinculoAutenticacionActor.PrincipalID =
				"per_otra234567890abcdefghijklmn"
		}},
		{"vinculo sin procedencia", func(d *decisionAutorizacionCanonicaV3) {
			d.VinculoAutenticacionActor.AutoridadEfectiva = ""
		}},
	}
	for _, caso := range mutaciones {
		t.Run(caso.nombre, func(t *testing.T) {
			mutado := reescribirDecisionCanonicaAtestacionV3Prueba(
				t,
				mensaje,
				caso.mutar,
			)
			if _, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(
				mutado,
			); !errors.Is(err, ErrParseoAtestacionAutorizacionV3Invalido) {
				t.Fatalf("mutación semántica aceptada: %v", err)
			}
		})
	}
}

func TestProyeccionAtestacionAutorizacionV3BloqueaCodecsYLogs(t *testing.T) {
	cabecera, decision, motivo, contexto := escenarioAtestacionAutorizacionV3Prueba(t)
	mensaje, _ := SerializarMensajeAtestacionAutorizacionV3(
		cabecera,
		decision,
		motivo,
		contexto,
	)
	proyeccion, err := ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(mensaje)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(proyeccion); !errors.Is(
		err,
		ErrSerializacionProyeccionAtestacionAutorizacionV3Prohibida,
	) {
		t.Fatalf("JSON no bloqueado: %v", err)
	}
	const marca = "[PROYECCION-VEC-AD-3-NOMINAL-NO-AUTORITATIVA-REDACTADA]"
	for _, texto := range []string{
		fmt.Sprint(proyeccion),
		fmt.Sprintf("%+v", proyeccion),
		fmt.Sprintf("%#v", proyeccion),
	} {
		if !strings.Contains(texto, marca) || strings.Contains(texto, "dec_0123456789") {
			t.Fatalf("formato filtró datos: %q", texto)
		}
	}
	var registro bytes.Buffer
	slog.New(slog.NewTextHandler(&registro, nil)).Info("prueba", "valor", proyeccion)
	if !strings.Contains(registro.String(), marca) ||
		strings.Contains(registro.String(), "dec_0123456789") {
		t.Fatalf("slog filtró datos: %s", registro.String())
	}
}

func FuzzParsearMensajeAtestacionAutorizacionV3NoAutoritativoNoEntraEnPanico(
	f *testing.F,
) {
	f.Add([]byte{})
	f.Add([]byte(EsquemaMensajeAtestacionAutorizacionV3))
	f.Fuzz(func(_ *testing.T, contenido []byte) {
		_, _ = ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(contenido)
	})
}

func reescribirDecisionCanonicaAtestacionV3Prueba(
	t *testing.T,
	mensaje []byte,
	mutar func(*decisionAutorizacionCanonicaV3),
) []byte {
	t.Helper()
	return reescribirDecisionAtestacionV3Prueba(
		t,
		mensaje,
		func(contenido []byte) []byte {
			var decision decisionAutorizacionCanonicaV3
			if err := json.Unmarshal(contenido, &decision); err != nil {
				t.Fatal(err)
			}
			mutar(&decision)
			mutado, err := json.Marshal(decision)
			if err != nil {
				t.Fatal(err)
			}
			return mutado
		},
	)
}

func reescribirDecisionAtestacionV3Prueba(
	t *testing.T,
	mensaje []byte,
	transformar func([]byte) []byte,
) []byte {
	t.Helper()
	lector := lectorHistoricoAtestacionAutorizacionV1{contenido: mensaje}
	lector.exigirBytes([]byte(EsquemaMensajeAtestacionAutorizacionV3))
	lector.exigirByte(0)
	_ = lector.leerUint16()
	_ = lector.leerTexto(128)
	_ = lector.leerTexto(512)
	_ = lector.leerTexto(512)
	inicioLongitud := lector.posicion
	longitud := lector.leerUint32()
	inicio := lector.posicion
	decision := lector.tomar(int(longitud))
	if lector.err != nil {
		t.Fatal("fixture VEC-AD-3 ilegible")
	}
	nuevaDecision := transformar(append([]byte(nil), decision...))
	resultado := make([]byte, 0, len(mensaje)+len(nuevaDecision)-len(decision))
	resultado = append(resultado, mensaje[:inicioLongitud]...)
	var longitudBinaria [4]byte
	binary.BigEndian.PutUint32(longitudBinaria[:], uint32(len(nuevaDecision)))
	resultado = append(resultado, longitudBinaria[:]...)
	resultado = append(resultado, nuevaDecision...)
	resultado = append(resultado, mensaje[inicio+len(decision):]...)
	binary.BigEndian.PutUint64(resultado[len(resultado)-8:], uint64(len(resultado)))
	return resultado
}

func sha256SumAtestacionV3Prueba(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
