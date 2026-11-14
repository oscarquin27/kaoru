// かおる, la lectura de «kaoru», en trazos del mismo grosor y remate que la K. Va sobre el wordmark
// como furigana. Son trazos y no texto para no depender de que el equipo tenga una fuente japonesa
// (Kaoru se instala sin conexión). Mismo dibujo que KANA en tools/boceto/js/boceto.js.
export function Kana({ className = "brand__kana" }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 88 24" aria-hidden="true">
      <path d="M4 9.2C7.5 8.7 11 8.3 14.2 8.4C15.8 8.5 16.2 10 15.8 12.5C15.3 15.5 14.3 18 12.2 19.8" />
      <path d="M9.8 4.2C9.5 9.5 8.2 15 5.2 20" />
      <path d="M17.8 7.4C19.3 9.1 20.2 11 20.6 13.2" />
      <g transform="translate(32 0)">
        <path d="M3.8 8.4H13.4" />
        <path d="M8.6 4V17.6C8.6 19.6 7.6 20.4 6.3 20.2C4.8 19.9 4.4 18.3 5.4 17C7.4 14.4 11 13.1 14.1 13.3C17.1 13.5 19 15.2 18.8 17.5C18.6 19.5 16.6 20.5 13.9 20.6" />
        <path d="M16.6 6.2C17.9 7 18.9 8 19.5 9.3" />
      </g>
      <g transform="translate(64 0)">
        <path d="M6.5 5.2H16.6L6 16.8C9 14 12.8 12.9 15.9 13.6C18.6 14.3 19.5 17.1 17.6 19C15.6 21 11.6 20.9 10.5 19.2C9.8 18 10.9 16.8 12.3 17.2C13.5 17.6 13.8 19.1 13 20.3" />
      </g>
    </svg>
  );
}
