// Nombres cortos para los tipos generados desde api/openapi.yaml (`npm run generate`).
import type { components } from './schema.d.ts'

type Schemas = components['schemas']

export type Me = Schemas['Me']
export type Role = Schemas['Role']
export type System = Schemas['System']
export type Workflow = Schemas['Workflow']
export type WorkflowTransition = Workflow['transitions'][number]
export type WorkflowState = Workflow['states'][number]
export type BlockingRule = Schemas['BlockingRule']
export type RolePermissions = Schemas['RolePermissions']
export type StateRef = Schemas['StateRef']
export type UserRef = Schemas['UserRef']
export type User = Schemas['User']
export type Route = Schemas['Route']
export type Severity = Schemas['Severity']
export type RiskLevel = Schemas['RiskLevel']
export type RunResult = Schemas['RunResult']
export type EntityType = Schemas['EntityType']
export type LinkKind = Schemas['LinkKind']
export type DocumentKind = Schemas['DocumentKind']
export type QualityCharacteristic = Schemas['QualityCharacteristic']

export type Change = Schemas['Change']
export type ChangeInput = Schemas['ChangeInput']
export type ChangePatch = Schemas['ChangePatch']
export type Requirement = Schemas['Requirement']
export type RequirementInput = Schemas['RequirementInput']
export type RequirementPatch = Schemas['RequirementPatch']
export type RequirementOverview = Schemas['RequirementOverview']
export type Link = Schemas['Link']
export type LinkInput = Schemas['LinkInput']
export type LinkPatch = Schemas['LinkPatch']
export type EntryContractItem = Schemas['EntryContractItem']
export type Risk = Schemas['Risk']
export type RiskInput = Schemas['RiskInput']
export type FunctionalCase = Schemas['FunctionalCase']
export type FunctionalCaseInput = Schemas['FunctionalCaseInput']
export type TestCase = Schemas['TestCase']
export type TestCaseInput = Schemas['TestCaseInput']
export type TestCaseSummary = Schemas['TestCaseSummary']
export type Run = Schemas['Run']
export type RunInput = Schemas['RunInput']
export type RunSummary = Schemas['RunSummary']
export type Exploration = Schemas['Exploration']
export type ExplorationInput = Schemas['ExplorationInput']
export type Defect = Schemas['Defect']
export type DefectInput = Schemas['DefectInput']
export type DefectPatch = Schemas['DefectPatch']
export type DefectSummary = Schemas['DefectSummary']
export type Document = Schemas['Document']
export type DocumentVersion = Schemas['DocumentVersion']
export type Comment = Schemas['Comment']
export type SearchHit = Schemas['SearchHit']
export type Matrix = Schemas['Matrix']
export type MatrixRow = Matrix['items'][number]
export type AuditEntry = Schemas['AuditEntry']
export type PageInfo = Schemas['PageInfo']

export type Page<T> = PageInfo & { items: T[] }
export interface Items<T> {
  items: T[]
}
export interface TransitionInput {
  to: string
  reason?: string
}
export interface AuditVerification {
  intact: boolean
  entries: number
  firstBrokenSeq?: number
}
