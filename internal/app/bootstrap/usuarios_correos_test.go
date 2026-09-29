package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	smtpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/smtp"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/i18n"
	core "vec-diputacion-granada/internal/vec/domain"
)

type smtpCorreosPrueba struct {
	mensajes []smtpct.Mensaje
	estado   smtpct.Estado
}

func (s *smtpCorreosPrueba) Enviar(_ context.Context, m smtpct.Mensaje) smtpct.Resultado {
	s.mensajes = append(s.mensajes, m)
	return smtpct.Resultado{Estado: s.estado}
}

type textosCorreosPrueba map[string]string

func (t textosCorreosPrueba) Message(_, clave string) (string, bool) {
	v, ok := t[clave]
	return v, ok
}

func textosCorreosCompletos() textosCorreosPrueba {
	return textosCorreosPrueba{
		claveAsuntoCodigoCorreo: "Código", claveCuerpoCodigoCorreo: "Su código es {codigo}; vale {minutos} minutos.",
		claveAsuntoAvisoCorreo: "Aviso", claveCuerpoAvisoCorreo: "Ha cambiado su correo.",
	}
}

func TestTransporteCorreosRedactaDesdeCatalogo(t *testing.T) {
	smtp := &smtpCorreosPrueba{estado: smtpct.AceptadoPorRelay}
	tr, err := nuevoTransporteCorreosPropiosDesarrollo(smtp, textosCorreosCompletos(), "VEC <avisos@dipgra.example>")
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC)
	tr.ahora = func() time.Time { return ahora }
	envio := "correo_envio:" + strings.Repeat("a", 32)
	ok := tr.EnviarCorreoPropio(context.Background(), usuariosports.MensajeCorreoPropio{EnvioRef: envio, Tipo: usuariosports.TipoEnvioCodigo, Destino: "a@example.org", Codigo: "12345678", VenceUTC: ahora.Add(time.Hour)})
	if !ok || len(smtp.mensajes) != 1 {
		t.Fatal("código no enviado")
	}
	m := smtp.mensajes[0]
	if m.Cuerpo != "Su código es 1234 5678; vale 60 minutos." || m.Asunto != "Código" || m.MessageID != "<correo-"+strings.Repeat("a", 32)+"@dipgra.example>" || !m.FechaOrigen.Equal(ahora) {
		t.Fatalf("mensaje mal redactado: %+v", m)
	}
	smtp.estado = smtpct.Indeterminado
	if tr.EnviarCorreoPropio(context.Background(), usuariosports.MensajeCorreoPropio{EnvioRef: envio, Tipo: usuariosports.TipoEnvioAviso, Destino: "a@example.org"}) {
		t.Fatal("resultado incierto contado como aceptado")
	}
	if smtp.mensajes[1].Cuerpo != "Ha cambiado su correo." || strings.Contains(smtp.mensajes[1].Cuerpo, "{") {
		t.Fatalf("aviso mal redactado: %+v", smtp.mensajes[1])
	}
	for _, malo := range []usuariosports.MensajeCorreoPropio{
		{EnvioRef: "otro", Tipo: usuariosports.TipoEnvioAviso, Destino: "a@example.org"},
		{EnvioRef: envio, Tipo: "otro", Destino: "a@example.org"},
		{EnvioRef: envio, Tipo: usuariosports.TipoEnvioCodigo, Destino: "a@example.org", Codigo: "12345678", VenceUTC: ahora},
	} {
		if tr.EnviarCorreoPropio(context.Background(), malo) {
			t.Fatalf("mensaje inválido enviado: %+v", malo)
		}
	}
	if len(smtp.mensajes) != 2 {
		t.Fatal("un mensaje inválido llegó al relay")
	}
	incompletos := textosCorreosCompletos()
	incompletos[claveCuerpoCodigoCorreo] = "Sin marcador"
	if _, err := nuevoTransporteCorreosPropiosDesarrollo(smtp, incompletos, "avisos@dipgra.example"); err == nil {
		t.Fatal("cuerpo sin {codigo} aceptado")
	}
	if _, err := nuevoTransporteCorreosPropiosDesarrollo(smtp, textosCorreosCompletos(), "sin-dominio"); err == nil {
		t.Fatal("remitente sin dominio aceptado")
	}
}

func TestCatalogoServidorTieneTextosDeCorreos(t *testing.T) {
	catalogo, err := i18n.LoadDir("../../../locales")
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		for _, clave := range []string{claveAsuntoCodigoCorreo, claveCuerpoCodigoCorreo, claveAsuntoAvisoCorreo, claveCuerpoAvisoCorreo} {
			texto, ok := catalogo.Message(idioma, clave)
			if !ok || strings.TrimSpace(texto) == "" {
				t.Fatalf("falta %s en %s", clave, idioma)
			}
		}
		if cuerpo, _ := catalogo.Message(idioma, claveCuerpoCodigoCorreo); !strings.Contains(cuerpo, "{codigo}") || !strings.Contains(cuerpo, "{minutos}") {
			t.Fatalf("cuerpo del código sin marcadores en %s", idioma)
		}
	}
}

func TestDescriptoresCorreosUnicosYOrdenados(t *testing.T) {
	d := descriptoresMaterialCorreosUsuariosDesarrollo()
	if len(d) != 12 {
		t.Fatalf("descriptores: %d", len(d))
	}
	if _, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(append(descriptoresMaterialPreferenciasUsuariosDesarrollo(), d...)); err != nil {
		t.Fatal("descriptores repetidos con preferencias")
	}
	for i, a := range accionesCorreosUsuarios {
		if !strings.Contains(d[i].Audiencia, "."+a.segmento+".interna_corporativa.") || !strings.Contains(d[i+6].Audiencia, "."+a.segmento+".externa_personal.") {
			t.Fatalf("orden de audiencias roto en %d", i)
		}
	}
	gobierno := map[string]bool{}
	for _, a := range audienciasConsumoGobiernoCTDesarrollo() {
		gobierno[a] = true
	}
	for _, a := range audienciasCorreosUsuariosDesarrollo() {
		if !gobierno[a] {
			t.Fatalf("audiencia %s fuera del gobierno", a)
		}
	}
	dep := &dependenciasCorreosUsuariosDesarrollo{}
	for i := range dep.materiales.lote {
		dep.materiales.lote[i] = &proveedorMaterialAltaContratacionTemporalDesarrollo{}
	}
	interna, ok := dep.materialesSuperficie(core.SuperficieAutenticacionInternaCorporativaV1)
	externa, ok2 := dep.materialesSuperficie(core.SuperficieAutenticacionExternaPersonalV1)
	if !ok || !ok2 || interna[0] != dep.materiales.lote[0] || externa[0] != dep.materiales.lote[6] {
		t.Fatal("materiales por superficie mal repartidos")
	}
	if _, ok := dep.materialesSuperficie(core.SuperficieAutenticacionAdministracionPrivilegiadaV1); ok {
		t.Fatal("superficie privilegiada admitida")
	}
}

func TestClavesCorreosDerivadasDistintas(t *testing.T) {
	var base [32]byte
	base[0] = 7
	f, err := nuevaFuenteClavesCorreosDesarrollo(&emisorKMSDesarrollo{claveEnvoltura: base})
	if err != nil {
		t.Fatal(err)
	}
	c := f.claves
	materiales := map[[32]byte]bool{c.CifradoActivo.Material: true, c.Igualdad.Material: true, c.SemanticaActiva.Material: true, c.CodigoActivo.Material: true}
	if len(materiales) != 4 || c.CifradoActivo.Material == claveContactoUsuarioDesarrollo(base) {
		t.Fatal("claves de correos repetidas o compartidas con contacto")
	}
	if _, err := nuevaFuenteClavesCorreosDesarrollo(&emisorKMSDesarrollo{}); err == nil {
		t.Fatal("KMS vacío aceptado")
	}
}
