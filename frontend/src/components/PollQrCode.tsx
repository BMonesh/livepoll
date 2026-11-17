import { useEffect, useState } from 'react'
import QRCode from 'qrcode'

export default function PollQrCode({ url }: { url: string }) {
  const [image, setImage] = useState('')
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let active = true
    QRCode.toDataURL(url, { width: 448, margin: 4, errorCorrectionLevel: 'M' }).then(
      data => { if (active) setImage(data) },
      () => { if (active) setFailed(true) },
    )
    return () => { active = false }
  }, [url])

  if (failed) return <p role="alert">Could not generate the QR code. You can still copy the link.</p>
  if (!image) return <p role="status">Generating QR code…</p>
  return <img className="qr-image" src={image} width="448" height="448" alt="Scan this QR code to open the poll" />
}
