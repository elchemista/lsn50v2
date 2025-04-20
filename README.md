# Dragino lsn50v2 Decoder

## Features

- **Zero external deps** – only uses Go’s standard library.
- **Work‑mode handlers** – each Dragino mode (0–5, 7–8) is its own `ModeHandler` implementation.
- **Battery, temperature, ADC, counter, weight, distance, humidity, illumination, DS18B20…** all decoded cleanly.
- **Easy to extend** – add new modes or tweak existing ones without touching the core decode logic.


## Installation

```bash
go get github.com/elchemista/lsn50v2
```

Or drop the `lsn50v2` folder straight into your project’s `pkg/` or wherever you like.


##  Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/elchemista/lsn50v2"
)

func main() {
    // Base64 payload from your Dragino device
    payload := "YOUR_BASE64_PAYLOAD_HERE"

    decoder := lsn50v2.NewDecoder()
    measurements, err := decoder.Decode(payload)
    if err != nil {
        log.Fatalf("Decode error: %v", err)
    }

    fmt.Println("Decoded measurements:")
    for _, m := range measurements {
        fmt.Printf(" • %s = %.3f\n", m.Name, m.Value)
    }
}
```


## Handling Custom Modes

If Dragino ever releases a new mode (or you want to tweak an existing one), it’s trivial:

1. Create a new type that implements `ModeHandler`.
2. Add your decoding logic in the `Decode(*Packet)` method.
3. Register it in `NewDecoder()`:

   ```go
   d.handlers[6] = myCustomMode6{}
   ```

That’s it—no giant switch statements to juggle.


## Contributing

Feel free to open issues or PRs! If you spot a bug, want a new mode, or just have ideas for making this decoder even better, I’d love to hear from you.


## License

This code is released under the MIT License. See [`LICENSE`](./LICENSE) for details.