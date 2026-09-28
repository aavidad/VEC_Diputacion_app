import { MENSAJES_PERSONAL_ES } from "../modulos/personal/i18n.js";
import { IDIOMA_ACTUAL, LOCALIZACION_ACTUAL } from "../../comun/idioma.js";

const prefijo = "organizacion_";
export const MENSAJES_ORGANIZACION_ES = Object.freeze(Object.fromEntries(
  Object.entries(MENSAJES_PERSONAL_ES).filter(([clave]) => clave.startsWith(prefijo)),
));

// Las advertencias conservan el alcance: preparar, conciliar o autenticar no
// acredita publicación, ocupación, autorización ni eficacia administrativa.
const EN = {
  importTab: "Import and reconcile", importTitle: "Governed establishment and RPT import",
  importStageInitial: "No batch", importStagePrepared: "Batch prepared", importStageConciliated: "Reconciliation recorded",
  importPrepareTitle: "Prepare source", importChoosePackage: "Choose JSON package", importPrepareAction: "Review preparation",
  importConciliateTitle: "Reconcile facts", importChooseDecisions: "Choose JSON decisions", importConciliateAction: "Review reconciliation",
  importPublishTitle: "Publish version", importPublishAction: "Publish",
  importPublishBlocked: "Publishing is disabled: this screen does not receive verified source, act, custody and publishing-authority evidence.",
  importHelp: "The JSON package must come from a governed source and contain a manifest and typed facts without actor or organisation identities. Preparation does not prove occupancy, vacancies or publication. Reconciliation uses decisions with their own evidence. Retry an uncertain response with the same key and bytes; reloading the page loses that in-memory retry.",
  importReady: "Choose a source package to prepare a batch.", importPackageReady: "Package loaded for review. No batch has been recorded yet.",
  importBadPackage: "The package does not comply with the Organisation manifest and facts contract.", importBadDecisions: "The decisions do not comply with the reconciliation contract.", importTooLarge: "The JSON exceeds the 8 MiB limit.",
  importReviewPrepare: "Preparation review", importReviewConciliate: "Reconciliation review", importConfirm: "Confirm operation", importCancel: "Return to review",
  importSending: "Recording operation…", importSaved: "Operation recorded", importRejected: "The server rejected the operation. Check the contract, catalogue and permissions.",
  importDenied: "You do not have permission for this operation.", importConflict: "Revision conflict. The batch will not be changed from this screen without retrieving a confirmed revision.",
  importUncertain: "The result could not be confirmed. You may only retry the same operation with the same key and body.", importRetryExact: "Retry exactly",
  importReceipt: "Receipt: {recibo}", importLot: "Batch: {lote}", importRevision: "Revision: {revision}", importSource: "Source: {fuente}",
  importSourceHash: "Source fingerprint: {huella}", importPackageHash: "JSON package fingerprint: {huella}", importVersion: "Version: {version}",
  importCatalogUnits: "Unit catalogue: {id} · v{version} · r{revision} · {huella}", importCatalogClasses: "Classification catalogue: {id} · v{version} · r{revision} · {huella}",
  importFactsOne: "{total} typed fact", importFactsMany: "{total} typed facts", importDecisionsOne: "{total} decision", importDecisionsMany: "{total} decisions",
  importOperation: "{fase} · key {clave}", importStatePreparation: "Preparation without publishing authority", importStatePending: "Reconciliation pending", importStateConciliated: "Reconciliation closed",
  importClassNodeOne: "{total} unit", importClassNodeMany: "{total} units", importClassTypeOne: "{total} post type", importClassTypeMany: "{total} post types",
  importClassAllocationOne: "{total} allocation", importClassAllocationMany: "{total} allocations", importClassPlazaOne: "{total} position", importClassPlazaMany: "{total} positions",
  importClassPostOne: "{total} individual post", importClassPostMany: "{total} individual posts", importClassLinkOne: "{total} link", importClassLinkMany: "{total} links",
  importFileReadError: "The selected file could not be read.", importReceiptDate: "Recorded (Europe/Madrid): {fecha}",
  documentTitle: "Organisation · Employee Portal", helpOpen: "?", tabsLabel: "Organisation queries", historyTab: "Historical establishment and RPT", catalogTab: "Unit catalogue",
  historyTitle: "Historical establishment and RPT", historyReadOnly: "Authorised query", historyPending: "No authorised query", historyDeniedPill: "Access denied",
  historyUnit: "Unit", historyChooseUnit: "All authorised units", historyValidDate: "Effective date", historyKnownAt: "Information known at (Madrid time)",
  historyDateFormat: "dd/mm/yyyy", historyDateTimeFormat: "dd/mm/yyyy HH:mm", historyVersionRPT: "RPT version (optional)", historyVersionPlantilla: "Establishment version (optional)",
  historySearch: "Query history", historyNeedUnit: "Loading unit catalogue…", historyReady: "Query pending", historyKind: "Show", historyUnits: "Units", historyTypes: "RPT post types",
  historyAllocations: "Allocations", historyPlazas: "Establishment positions", historyPosts: "Individual posts", historyLinks: "Position–post links", historyMore: "Show more results",
  historyLoading: "Querying history…", historyLoaded: "Historical query completed. Review coverage, version and results.", historyDenied: "You do not have permission for this query.",
  historyError: "The history could not be queried. Check the filters and try again.", historyInvalid: "Choose valid dates for the query.",
  historyEmpty: "The source returns no rows for this cut. Check coverage before interpreting the empty result.", historyNoSource: "No source data", historyPartial: "Partial coverage", historyComplete: "Complete coverage",
  historyCount: "{total} rows on this page", historyPageScope: "Rows on this page and coverage declared by each source", historyEffective: "Effective: {fecha}",
  historyKnown: "Known: {fecha} (Europe/Madrid)", historyRPT: "RPT: {version}", historyPlantilla: "Establishment: {version}", historyNoVersion: "No version supplied", historyReceipt: "Receipt: {referencia}",
  historyCode: "Source code", historyName: "Name", historyType: "Type", historyParent: "Unit / parent unit", historySource: "Source", historyQuantity: "Quantity",
  historyClass: "Classification", historyStructural: "Structural status", historyPlazaCode: "Position", historyPostCode: "Post", historyNotProvided: "Not provided",
  eyebrow: "Reference organisation · Preparation", title: "Reference organisation in preparation", intro: "Reference organisational structure query and maintenance for Human Resources.",
  back: "Back to Employee Portal", provenanceDetail: "Data provenance and scope", traceability: "Catalogue version and traceability", noticeTitle: "Catalogue provenance",
  notice: "Editing prepares a catalogue change; it does not prove occupants, functional reporting or ratification permissions.", catalogId: "Catalogue", version: "Version", revision: "Revision", fingerprint: "SHA-256 fingerprint", status: "Status",
  tableTitle: "Organisational units", filterText: "Filter by text", filterType: "Filter by type", allTypes: "All types", delegacion: "Group", centro: "Centre", puesto: "Position of responsibility",
  tableCaption: "Units in the queried version", type: "Type", code: "Code / key", name: "Name", adscription: "Parent unit", page: "PDF page", actions: "Actions",
  newUnit: "New unit", edit: "Edit", localChange: "Local change", sourceLabel: "Declared source:", sourceLink: "View transparency information",
  note: "New keys are technical and do not identify people or invent official codes.", loading: "Loading organisation…", error: "The organisation could not be loaded.", retry: "Retry",
  empty: "No units match the filters.", count: "{visible} of {total} units", draft: "Draft", editorTitle: "Prepare creation or edit",
  editorHelper: "This form stores name, type and parent unit. Review the summary before confirming; it does not enable ratification.", label: "Name", parent: "Parent unit", noParent: "No parent unit", reason: "Reason",
  cancel: "Cancel", review: "Review change", confirm: "Confirm change", saving: "Saving…", retryExact: "Retry the same change", reloadReview: "Reload and review again",
  conflict: "The catalogue changed. The form is preserved; reload before confirming.", uncertain: "It could not be determined whether the change was recorded. Keep the same key and body to retry.",
  saved: "Change recorded", receiptRef: "Reference", receiptDate: "Date", reloadError: "The receipt is preserved, but the catalogue could not be reloaded.",
  rejected: "Change rejected without saving. Check the data, parent unit and your permissions before confirming again.",
};

export const MENSAJES_ORGANIZACION_EN = Object.freeze(Object.fromEntries(
  Object.entries(MENSAJES_ORGANIZACION_ES).map(([clave, texto]) => [clave, EN[clave.slice(prefijo.length)] ?? texto]),
));

const claves = Object.freeze(Object.keys(MENSAJES_ORGANIZACION_ES));
function comprobarCatalogo(catalogo) {
  if (!catalogo || typeof catalogo !== "object" || claves.some((clave) => typeof catalogo[clave] !== "string" || catalogo[clave] === "")) {
    throw new Error("catálogo i18n de Organización incompleto");
  }
}
comprobarCatalogo(MENSAJES_ORGANIZACION_EN);

export function crearTraductorOrganizacion(idioma = IDIOMA_ACTUAL) {
  const catalogo = idioma === "en" ? MENSAJES_ORGANIZACION_EN : MENSAJES_ORGANIZACION_ES;
  return (clave, variables = {}) => {
    const claveCompleta = clave.startsWith(prefijo) ? clave : `${prefijo}${clave}`;
    if (!claves.includes(claveCompleta)) throw new Error(`clave i18n de Organización desconocida: ${claveCompleta}`);
    return catalogo[claveCompleta].replace(/\{([a-z_]+)\}/gu, (_texto, variable) => String(variables[variable] ?? ""));
  };
}

export const localizacionOrganizacion = () => LOCALIZACION_ACTUAL;
