// ==========================================
// FOLLOW-THE-FUCKING-MONEY (FTFM)
// Constraints de Unicidade e Índices de Busca
// ==========================================

// 1. Constraints de Chaves Primárias
CREATE CONSTRAINT ON (c:Candidate) ASSERT c.sq_candidate IS UNIQUE;
CREATE CONSTRAINT ON (p:Person) ASSERT p.id IS UNIQUE;
CREATE CONSTRAINT ON (e:Company) ASSERT e.cnpj IS UNIQUE;
CREATE CONSTRAINT ON (cp:PublicContract) ASSERT cp.id_contract IS UNIQUE;
CREATE CONSTRAINT ON (o:PublicAgency) ASSERT o.ug IS UNIQUE;
CREATE CONSTRAINT ON (ep:ParliamentaryAmendment) ASSERT ep.code IS UNIQUE;

// 2. Índices de Busca B-Tree
CREATE INDEX ON :Company(cnpj_root);
CREATE INDEX ON :Company(state);
CREATE INDEX ON :Person(masked_cpf);
CREATE INDEX ON :Candidate(year);
CREATE INDEX ON :Candidate(ballot_name);
CREATE INDEX ON :PublicContract(year);

// 3. Índices de Relacionamentos
CREATE INDEX ON :DOOU_PARA(year);
CREATE INDEX ON :DOOU_PARA(amount);
CREATE INDEX ON :FORNECEU_PARA(year);
CREATE INDEX ON :ASSINOU_CONTRATO(amount);
