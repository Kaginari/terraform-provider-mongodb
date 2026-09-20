# terraform-provider-mongodb — Isekai world

This directory is reincarnated per the Isekai convention. Before doing any work this session:

1. Read `.isekai/isekai.md` in full — it governs how work here is done, and every creature
   (including Rimuru, the session agent) reads it before working.
2. Skim the last 5–10 entries of `.isekai/log.md` for what the world already learned.

This applies **every session, including immediately after `/clear`.** The convention's whole
premise — "the world remembers in documents, because sessions forget" — depends on this file
being the thing that actually reconnects a fresh session to what was written down. Without it,
the mid-session stress-relief instrument (`.isekai/tools/context-check.sh`) only does half its
job: it tells a stressed session to write to `log.md`, but nothing told the next session to
read it back. This file is that missing half.
