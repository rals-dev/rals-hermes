// Body frames per pose (docs/decisions/024): the figure without hair or
// accessories, 16×24, black outline, two-tone shading. The head is bald
// here — hair and accessories are layers (hair.ts, accessories.ts) stacked
// on top by compose.ts. On stride frames the whole upper body is drawn one
// row lower (headDy = 1) so the walk bobs; layers follow via headDy.

export const SPRITE_W = 16
export const SPRITE_H = 24
/** Seated poses draw rows 0–18 only (waist on row 18); the desk hides the rest. */
export const SEATED_GROUND = 19

export type AgentPose =
  | 'seatedIdle' | 'seatedTyping' | 'seatedPhone' | 'seatedError'
  | 'stand' | 'talk' | 'walkFront' | 'walkBack' | 'walkSide'

export type View = 'front' | 'back' | 'side'

export interface BodyFrame {
  rows: readonly string[]
  /** How far the head (and every layer drawn on it) sits below its standing position. */
  headDy: 0 | 1
}

const SEATED_IDLE: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_IDLE_RAISED: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..osocccccCoso..', // 15
  '..ococcccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_TYPING_L: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..occcccccscco..', // 16
  '...oospppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_TYPING_R: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..occsccccccco..', // 16
  '...oopppppsoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_PHONE: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..ossssssssskk..', // 4
  '..ossssssssskko.', // 5
  '..ossesssseSkko.', // 6
  '..ossssssssskk..', // 7
  '...osssSSsssss..', // 8
  '....oSssssSo.co.', // 9
  '.....ooSSoo..co.', // 10
  '...ooccccccooco.', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_PHONE_UP: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..ossssssssskk..', // 3
  '..ossssssssskko.', // 4
  '..ossssssssskko.', // 5
  '..ossesssseSkk..', // 6
  '..osssssssssss..', // 7
  '...osssSSsssoco.', // 8
  '....oSssssSo.co.', // 9
  '.....ooSSoo..co.', // 10
  '...ooccccccooco.', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const SEATED_ERROR: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '.osssssssssssso.', // 3
  '.ocssssssssssco.', // 4
  '.ocssssssssssco.', // 5
  '.ocssesssseSsco.', // 6
  '.ocssssssssssco.', // 7
  '.ocosssSSsssoco.', // 8
  '.oc.oSssssSo.co.', // 9
  '.oc..ooSSoo..co.', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..ococcccccoco..', // 16
  '...ooppppppoo...', // 17
  '....oPPPPPPo....', // 18
  '................', // 19
  '................', // 20
  '................', // 21
  '................', // 22
  '................', // 23
]

const STAND: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossesssseSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const STAND_BLINK: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..ossSssssSSso..', // 6
  '..osssssssssso..', // 7
  '...osssSSssso...', // 8
  '....oSssssSo....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const TALK_LOW: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo......', // 10
  '....occcccoo....', // 11
  '....occccCoso...', // 12
  '....occcCCCo....', // 13
  '....occccCo.....', // 14
  '....occccCo.....', // 15
  '....occccCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const TALK_HIGH: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo.so...', // 10
  '....occcccCo....', // 11
  '....occcCCo.....', // 12
  '....occccCo.....', // 13
  '....occccCo.....', // 14
  '....occccCo.....', // 15
  '....occccCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const WALK_FRONT_STEP_L: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..ossesssseSso..', // 7
  '..osssssssssso..', // 8
  '...osssSSssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....oPPoobbo....', // 21
  '....obbooooo....', // 22
  '....ooo.........', // 23
]

const WALK_FRONT_STEP_R: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..ossesssseSso..', // 7
  '..osssssssssso..', // 8
  '...osssSSssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....obbooPPo....', // 21
  '....ooooobbo....', // 22
  '.........ooo....', // 23
]

const WALK_BACK_STEP_L: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '..osssssssssso..', // 8
  '...osssssssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....oPPoobbo....', // 21
  '....obbooooo....', // 22
  '....ooo.........', // 23
]

const STAND_BACK: readonly string[] = [
  '....oooooooo....', // 0
  '...osssssssso...', // 1
  '..osssssssssso..', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '...osssssssso...', // 8
  '....osssssso....', // 9
  '.....ooSSoo.....', // 10
  '...ooccccccoo...', // 11
  '..occcccccccco..', // 12
  '..occcccccccCo..', // 13
  '..ococcccccoCo..', // 14
  '..ococccccCoCo..', // 15
  '..osoccccccoso..', // 16
  '...ooppppppoo...', // 17
  '....oppppppo....', // 18
  '....oppooppo....', // 19
  '....oppooppo....', // 20
  '....oPPooPPo....', // 21
  '....obboobbo....', // 22
  '....ooo..ooo....', // 23
]

const WALK_BACK_STEP_R: readonly string[] = [
  '................', // 0
  '....oooooooo....', // 1
  '...osssssssso...', // 2
  '..osssssssssso..', // 3
  '..osssssssssso..', // 4
  '..osssssssssso..', // 5
  '..osssssssssso..', // 6
  '..osssssssssso..', // 7
  '..osssssssssso..', // 8
  '...osssssssso...', // 9
  '....oSssssSo....', // 10
  '.....ooSSoo.....', // 11
  '...ooccccccoo...', // 12
  '..occcccccccco..', // 13
  '..occcccccccCo..', // 14
  '..ococcccccoCo..', // 15
  '..ococccccCoCo..', // 16
  '..osoccccccoso..', // 17
  '...ooppppppoo...', // 18
  '....oppppppo....', // 19
  '....oppooppo....', // 20
  '....obbooPPo....', // 21
  '....ooooobbo....', // 22
  '.........ooo....', // 23
]

const WALK_SIDE_STRIDE_FWD: readonly string[] = [
  '................', // 0
  '.....oooooo.....', // 1
  '....osssssso....', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssssso...', // 5
  '...osssssseso...', // 6
  '...osssssssso...', // 7
  '...ossssssssso..', // 8
  '....ossssssSo...', // 9
  '.....oSsssso....', // 10
  '......oSSo......', // 11
  '....occccco.....', // 12
  '....occccCo.....', // 13
  '....occcCCoo....', // 14
  '....occccCCso...', // 15
  '....occccCoo....', // 16
  '....occccCo.....', // 17
  '....opppppo.....', // 18
  '...opp...ppo....', // 19
  '..opp.....ppo...', // 20
  '..opp.....ppo...', // 21
  '.obb......bbbo..', // 22
  '.oooo....ooooo..', // 23
]

const STAND_SIDE: readonly string[] = [
  '.....oooooo.....', // 0
  '....osssssso....', // 1
  '...osssssssso...', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssseso...', // 5
  '...osssssssso...', // 6
  '...ossssssssso..', // 7
  '....ossssssSo...', // 8
  '.....oSsssso....', // 9
  '......oSSo......', // 10
  '....occccco.....', // 11
  '....occCcCo.....', // 12
  '....occCcCo.....', // 13
  '....occCcCo.....', // 14
  '....occCcCo.....', // 15
  '....occscCo.....', // 16
  '....opppppo.....', // 17
  '.....opppo......', // 18
  '.....opppo......', // 19
  '.....opppo......', // 20
  '.....oPPPo......', // 21
  '.....obbbbo.....', // 22
  '.....oooooo.....', // 23
]

const WALK_SIDE_STRIDE_BACK: readonly string[] = [
  '................', // 0
  '.....oooooo.....', // 1
  '....osssssso....', // 2
  '...osssssssso...', // 3
  '...osssssssso...', // 4
  '...osssssssso...', // 5
  '...osssssseso...', // 6
  '...osssssssso...', // 7
  '...ossssssssso..', // 8
  '....ossssssSo...', // 9
  '.....oSsssso....', // 10
  '......oSSo......', // 11
  '....occccco.....', // 12
  '....occccCo.....', // 13
  '....ocCccCo.....', // 14
  '....oCcccCo.....', // 15
  '...osccccCo.....', // 16
  '....occccCo.....', // 17
  '....opppppo.....', // 18
  '...opp...ppo....', // 19
  '..opp.....ppo...', // 20
  '..opp.....ppo...', // 21
  '.obb......bbbo..', // 22
  '.oooo....ooooo..', // 23
]

export const BODY: Readonly<Record<AgentPose, readonly BodyFrame[]>> = {
  seatedIdle: [{ rows: SEATED_IDLE, headDy: 0 }, { rows: SEATED_IDLE_RAISED, headDy: 0 }],
  seatedTyping: [{ rows: SEATED_TYPING_L, headDy: 0 }, { rows: SEATED_TYPING_R, headDy: 0 }],
  seatedPhone: [{ rows: SEATED_PHONE, headDy: 0 }, { rows: SEATED_PHONE_UP, headDy: 0 }],
  seatedError: [{ rows: SEATED_ERROR, headDy: 0 }],
  stand: [{ rows: STAND, headDy: 0 }, { rows: STAND_BLINK, headDy: 0 }],
  talk: [{ rows: TALK_LOW, headDy: 0 }, { rows: TALK_HIGH, headDy: 0 }],
  walkFront: [{ rows: WALK_FRONT_STEP_L, headDy: 1 }, { rows: STAND, headDy: 0 }, { rows: WALK_FRONT_STEP_R, headDy: 1 }, { rows: STAND, headDy: 0 }],
  walkBack: [{ rows: WALK_BACK_STEP_L, headDy: 1 }, { rows: STAND_BACK, headDy: 0 }, { rows: WALK_BACK_STEP_R, headDy: 1 }, { rows: STAND_BACK, headDy: 0 }],
  walkSide: [{ rows: WALK_SIDE_STRIDE_FWD, headDy: 1 }, { rows: STAND_SIDE, headDy: 0 }, { rows: WALK_SIDE_STRIDE_BACK, headDy: 1 }, { rows: STAND_SIDE, headDy: 0 }],
}

export const VIEW_OF: Readonly<Record<AgentPose, View>> = {
  seatedIdle: 'front',
  seatedTyping: 'front',
  seatedPhone: 'front',
  seatedError: 'front',
  stand: 'front',
  talk: 'side',
  walkFront: 'front',
  walkBack: 'back',
  walkSide: 'side',
}
