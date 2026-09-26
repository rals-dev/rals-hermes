// Figures that aren't a persona's body: the empty chair shown for an
// offline agent in the grid view, and the small helper figure drawn beside
// a desk for a same-profile sub-agent (in its owner's hair and shirt).

export const CHAIR: readonly string[] = [
  '................', // 0
  '................', // 1
  '................', // 2
  '................', // 3
  '................', // 4
  '................', // 5
  '................', // 6
  '....oooooooo....', // 7
  '....oQQQQQQo....', // 8
  '....oqqqqqqo....', // 9
  '....oqqqqqqo....', // 10
  '....oqqqqqqo....', // 11
  '....oqqqqqqo....', // 12
  '....oqqqqqqo....', // 13
  '....oqqqqqqo....', // 14
  '....oqqqqqqo....', // 15
  '....oqqqqqqo....', // 16
  '..oQQQQQQQQQQo..', // 17
  '..oqqqqqqqqqqo..', // 18
  '..oqqqqqqqqqqo..', // 19
  '..oooooooooooo..', // 20
  '......oqqo......', // 21
  '...oooooooooo...', // 22
  '...o...oo...o...', // 23
]


/** 8×12 helper figure; wears its owner's hair (h) and shirt (c/C) colours. */
export const HELPER: readonly string[] = [
  '..oooo..', // 0
  '.ohhhho.', // 1
  'ohsssSho', // 2
  'ohesseho', // 3
  '.osssso.', // 4
  '..oSSo..', // 5
  '.occcco.', // 6
  'occcccCo', // 7
  'oscccCso', // 8
  '.oppppo.', // 9
  '.opoopo.', // 10
  '.oboobo.', // 11
]
