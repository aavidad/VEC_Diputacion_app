package bootstrap

import (
	"context"
	"errors"
	"log"
	"time"

	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

var errParametrosAvisosBolsaNoPublicados = errors.New("bootstrap: parametros de avisos de Bolsa no publicados; instale Bolsa 000041")

// componerParametrosAvisosBolsaDesarrollo publica en Bolsa (000041) los
// parámetros de avisos y marcas del catálogo de reglas (b16, b17 y b19) y,
// solo si la base los acepta, activa la bandeja v2 y las marcas del cuadro y
// la ficha. Sin catálogo, o sin esas reglas operativas, no cambia nada: rige
// la bandeja de siempre. Con ellas, una base sin la migración o un catálogo
// mal formado impiden arrancar en lugar de ignorar la configuración.
func componerParametrosAvisosBolsaDesarrollo(ctx context.Context, resolutor *reglas.Resolutor, fuente *fuenteConstituidaRRHHDesarrollo) error {
	if fuente == nil || fuente.parametros == nil || fuente.consultaAvisos == nil || !resolutor.Disponible() {
		return nil
	}
	if ctx == nil {
		return errParametrosAvisosBolsaNoPublicados
	}
	publicacion, hay, err := reglasbolsa.PoliticaAvisos(ctx, resolutor)
	if err != nil {
		return errors.Join(errReglasEjemploNoValidas, err)
	}
	if !hay {
		return nil
	}
	version, err := fuente.parametros.PublicarPoliticaAvisos(ctx, publicacion)
	if err != nil {
		log.Printf("bolsa: parametros de avisos no publicados (¿falta Bolsa 000041?)")
		return errors.Join(errParametrosAvisosBolsaNoPublicados, err)
	}
	fuente.consultaAvisos.ActivarParametros()
	fuente.marcas = fuente.parametros
	if intentos := reglasbolsa.NuevosIntentosContacto(resolutor); intentos.Configurada() {
		fuente.intentos = intentos
	}
	fuente.invalidar()
	log.Printf("bolsa: parametros de avisos del catalogo publicados; version=%d", version)
	return nil
}

// cargarMarcasBase prepara el conjunto para recibir las marcas. Sin marcas
// compuestas el mapa queda nulo y la salida no las incluye.
func (f *fuenteConstituidaRRHHDesarrollo) cargarMarcasBase(ctx context.Context, datos *datasetBolsasRRHHDesarrollo) error {
	if f.marcas == nil {
		return nil
	}
	datos.Marcas = map[string]dominiobolsa.MarcasParticipacion{}
	if f.intentos == nil {
		return nil
	}
	politica, _, configurada, err := f.intentos.PoliticaIntentosTelefonicos(ctx)
	if err != nil || (configurada && politica.Validar() != nil) {
		return errors.Join(ErrComposicionDesarrolloIncompleta, err)
	}
	if configurada {
		datos.PoliticaIntentos = &politica
	}
	return nil
}

func (f *fuenteConstituidaRRHHDesarrollo) cargarMarcas(ctx context.Context, datos *datasetBolsasRRHHDesarrollo, bolsaRef string) error {
	if f.marcas == nil {
		return nil
	}
	marcas, err := f.marcas.MarcasParticipaciones(ctx, bolsaRef, f.ahora())
	if err != nil {
		return err
	}
	for referencia, marca := range marcas {
		datos.Marcas[referencia] = marca
	}
	return nil
}

// salidaMarcas rotula una candidatura: ya presta servicios (b16), en
// revisión (duda 18) y encadenamiento (b17). La baja propuesta por intentos
// agotados sale de los contactos ya leídos y de la política del catálogo.
func (h *bolsasRRHHDesarrolloDatos) salidaMarcas(participacion, estado string) map[string]any {
	marca := h.datos.Marcas[participacion]
	salida := map[string]any{"presta_servicios": nuloBootstrap(marca.PrestaServicios), "en_revision": nuloBootstrap(marca.EnRevision), "encadenamiento": nil}
	if marca.EnRevision == "" && estado != dominiobolsa.SituacionExcluido && h.bajaPropuesta(participacion) {
		salida["en_revision"] = dominiobolsa.RevisionBajaPropuesta
	}
	if marca.EncadenamientoDias > 0 {
		salida["encadenamiento"] = map[string]any{"dias_acumulados": marca.EncadenamientoDias, "umbral_meses": marca.EncadenamientoUmbralMeses, "ventana_meses": marca.EncadenamientoVentanaMeses}
	}
	return salida
}

// bajaPropuesta aplica al último llamamiento telefónico de la participación
// el mismo cálculo que la ficha de intentos (b02 y b03).
func (h *bolsasRRHHDesarrolloDatos) bajaPropuesta(participacion string) bool {
	if h.datos.PoliticaIntentos == nil {
		return false
	}
	var propios []dominiobolsa.ContactoParticipacion
	ultimo, llamamiento := time.Time{}, ""
	for _, c := range h.datos.Contactos {
		if c.ParticipacionRef != participacion {
			continue
		}
		propios = append(propios, c)
		if c.Canal == dominiobolsa.CanalContactoTelefono && c.LlamamientoRef != "" && c.Instante.After(ultimo) {
			ultimo, llamamiento = c.Instante, c.LlamamientoRef
		}
	}
	if llamamiento == "" {
		return false
	}
	politica := *h.datos.PoliticaIntentos
	return dominiobolsa.EstadoIntentos(politica, dominiobolsa.ResumirIntentosTelefonicos(politica, llamamiento, propios)).BajaPropuesta
}
