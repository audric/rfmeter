# Testing Meshtastic devices with rfmeter

A short field guide for measuring the transmit power of a Meshtastic
(LoRa) node with the USB RF Power Meter and this app.

> [!CAUTION]
> **Never connect a transmitting radio directly to the meter.** A
> Meshtastic node can put out **+20 to +30 dBm (0.1 W to 1 W)**. The
> meter's safe input ceiling is **0 dBm (1 mW)** — feeding it a watt
> will destroy it instantly. **Always insert an attenuator** between
> the radio and the meter.

## Quick start

1. Pick an attenuator so the meter sees roughly **−20 to −5 dBm**
   (see [Attenuator examples](#attenuator-examples)).
2. Wire it inline: **node → attenuator → meter**. With the antenna
   removed and everything in coax, the meter terminates the chain and
   nothing radiates.
3. In the app, **set a band to your LoRa frequency** (e.g. 868 or
   915 MHz) — edit any of F3–F10's freq field and Apply.
4. Press **F2 (Attenuator helper)**, enter the attenuation in dB, and
   Apply — the app writes it as the band offset so the readout shows
   the radio's *true* output, not the pad's output.
5. Trigger a transmission and read the power.

## Why an attenuator is mandatory

The meter reports power on a log scale (dBm). An attenuator simply
subtracts a fixed number of dB:

```
P_meter (dBm) = P_node (dBm) − attenuation (dB)
```

So a +30 dBm (1 W) node behind a 40 dB pad lands at −10 dBm at the
meter — well inside its range, with headroom. The app then *adds* the
40 dB back as the active band's offset, so the on-screen value reads
+30 dBm again.

Two things to size for:

- **Attenuation (dB)** — bring the level under 0 dBm with margin.
- **Power rating (W)** — the pad must absorb the *full* node output.
  A pocket SMA attenuator rated for 2 W is fine for a 1 W node; a
  0.5 W pad would burn out. Match the [W ↔ dBm table](#w--dbm-reference)
  to your hardware.

## W ↔ dBm reference

`dBm = 10 · log₁₀(P / 1 mW)`. Every **+10 dB is ×10 power**; **+3 dB ≈ ×2**.

| dBm | Power | Typical meaning |
|----:|------:|---|
| +30 | 1 W | LoRa legal max in some regions (US915 PtP) |
| +27 | 500 mW | High-power node (PA module, e.g. RAK + amp) |
| +24 | 250 mW | |
| +22 | 158 mW | Common SX1262 firmware max |
| +20 | 100 mW | EU868 ERP-ish ceiling; frequent default |
| +17 | 50 mW | |
| +14 | 25 mW | EU868 10 % duty-cycle sub-band limit |
| +13 | 20 mW | |
| +10 | 10 mW | |
| +7 | 5 mW | |
| +3 | 2 mW | |
| **0** | **1 mW** | **meter safe-input ceiling** |
| −3 | 500 µW | |
| −10 | 100 µW | good measurement target |
| −20 | 10 µW | good measurement target |
| −30 | 1 µW | |
| −40 | 100 nW | |
| −50 | 10 nW | near noise floor |

## Attenuator examples

Goal: land the meter in its sweet spot (**≈ −20 to −5 dBm**) with a
little margin below the 0 dBm ceiling. Pick the *next pad up* if your
node's power is uncertain — too much attenuation only costs you a few
dB of dynamic range; too little costs you the meter.

| Node output | Use | Meter sees | Notes |
|---|---|---|---|
| +30 dBm (1 W) | 40 dB | −10 dBm | 30 dB would sit *at* the ceiling — no margin. Use 40 dB. |
| +27 dBm (500 mW) | 40 dB | −13 dBm | |
| +22 dBm (158 mW) | 30 dB | −8 dBm | |
| +20 dBm (100 mW) | 30 dB | −10 dBm | |
| +17 dBm (50 mW) | 20 dB | −3 dBm | thin margin; 30 dB → −13 dBm is safer |
| +14 dBm (25 mW) | 20 dB | −6 dBm | |
| ≤ +10 dBm | 10–20 dB | ≤ −10 dBm | still pad it; don't trust "low power" settings |

Stacking pads adds dB: a 20 dB + 20 dB chain = 40 dB. Account for the
small insertion loss of cables/adapters (≈ 0.5–1 dB at these
frequencies) if you need accuracy — or just calibrate it out (below).

**Power rating:** the *first* pad in the chain eats the full node
power. For a 1 W node use a pad rated ≥ 2 W; downstream pads only see
the already-reduced level.

### Calibrating the offset

Two ways to make the readout show the node's true output:

- **Quick:** F2 → enter the total attenuation (e.g. `40`) → Apply.
  The app stores it as the active band's offset.
- **Precise:** feed a *known* source (or the node at a known setting)
  through your exact pad/cable chain, then nudge the band offset until
  the readout matches the known value. This folds in cable loss,
  connector loss, and pad tolerance in one shot.

## Picking the band / frequency

Meshtastic runs in region-specific ISM bands. Set a calibration band
(F3–F10) to the centre of yours so the meter applies the right
frequency correction:

| Region preset | Frequency to enter |
|---|---|
| EU_868 | 869 MHz |
| US / ANZ (915) | 915 MHz |
| CN_470 | 470 MHz |
| IN_865 | 865 MHz |
| EU_433 / 433-band | 433 MHz |

Edit the freq field of any band button, Apply, then select it before
measuring. (Factory defaults are 100–6000 MHz; none land exactly on
the LoRa bands, so editing one is expected.)

## Triggering a transmission

LoRa traffic is bursty — the meter shows brief spikes, not a steady
carrier. To get a readable burst:

- Send a message / position from the device or the Meshtastic app.
- Or enable a periodic broadcast (e.g. position or telemetry interval)
  so packets repeat while you watch the plot.
- Watch the **peak** on the time-series plot (Space pauses it to read
  a spike); the big readout follows the latest sample, the histogram
  shows the distribution of recent bursts.
- For a sustained signal, use a CW/test-tone mode if your firmware
  exposes one — otherwise read the packet peaks.

Log a session to CSV with **F11** and grab a **PNG snapshot** with
**F12** for your test notes.

## Sanity checklist

- [ ] Attenuator inline *before* powering the radio's TX — never
      hot-plug the meter onto a live transmitter.
- [ ] Pad's power rating ≥ node output.
- [ ] Band frequency matches the LoRa region.
- [ ] Total attenuation entered as the band offset (F2).
- [ ] Meter reading lands well under 0 dBm with the pad in place.
