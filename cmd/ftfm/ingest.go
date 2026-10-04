package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/l-eduardo/follow-the-fucking-money/internal/domain"
	"github.com/l-eduardo/follow-the-fucking-money/internal/pipeline/extractor"
	"github.com/l-eduardo/follow-the-fucking-money/internal/pipeline/worker"
	"github.com/l-eduardo/follow-the-fucking-money/internal/storage/graph"
	"github.com/l-eduardo/follow-the-fucking-money/pkg/streamutil"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	tseYear      int
	sourcePath   string
	batchSize    int
	workersCount int
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingestão e processamento em lote de dados públicos para o grafo",
}

func formatNumber(n int) string {
	in := strconv.Itoa(n)
	out := make([]byte, 0, len(in)+len(in)/3)
	remainder := len(in) % 3
	if remainder > 0 {
		out = append(out, in[:remainder]...)
		if len(in) > remainder {
			out = append(out, '.')
		}
	}
	for i := remainder; i < len(in); i += 3 {
		out = append(out, in[i:i+3]...)
		if i+3 < len(in) {
			out = append(out, '.')
		}
	}
	return string(out)
}

func printProgress(category string, count int, start time.Time) {
	elapsed := time.Since(start).Seconds()
	rate := 0.0
	if elapsed > 0 {
		rate = float64(count) / elapsed
	}
	fmt.Printf("\r  ⏳ [%s] %s registros gravados | ~%.0f reg/seg", category, formatNumber(count), rate)
}

func printDone(category string, count int, start time.Time) {
	elapsed := time.Since(start).Seconds()
	rate := 0.0
	if elapsed > 0 {
		rate = float64(count) / elapsed
	}
	fmt.Printf("\r  ✓ [%s] %s registros gravados no grafo com sucesso! (em %.1fs | ~%.0f reg/s)\n", category, formatNumber(count), elapsed, rate)
}

var ingestTSECmd = &cobra.Command{
	Use:   "tse",
	Short: "Ingere dados eleitorais do TSE (candidaturas, bens, receitas e despesas)",
	Example: `  ftfm ingest tse --year 2022 --source ./downloads/tse/consulta_cand_2022.zip
  ftfm ingest tse --year 2022 --source ./downloads/tse/2022/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		svc, err := graph.NewService(cfg.GraphURI, cfg.GraphUser, cfg.GraphPass, cfg.DatabaseName)
		if err != nil {
			return fmt.Errorf("falha ao conectar no Graph DB: %w", err)
		}
		defer svc.Close(ctx)

		fmt.Printf("\n🚀 Iniciando Ingestão de Dados Eleitorais TSE (%d)\n", tseYear)
		fmt.Printf("   Origem: %s | Batch Size: %d | Workers: %d\n\n", sourcePath, batchSize, workersCount)

		start := time.Now()
		totalProcessed := 0

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return fmt.Errorf("caminho de origem não encontrado: %w", err)
		}

		var filesToProcess []string
		if fileInfo.IsDir() {
			entries, _ := os.ReadDir(sourcePath)
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".zip") || strings.HasSuffix(e.Name(), ".csv") {
					filesToProcess = append(filesToProcess, filepath.Join(sourcePath, e.Name()))
				}
			}
		} else {
			filesToProcess = append(filesToProcess, sourcePath)
		}

		for _, filePath := range filesToProcess {
			fileName := filepath.Base(filePath)

			if strings.Contains(fileName, "consulta_cand") {
				fmt.Printf("🏛️  [Candidaturas & Partidos] Processando %s...\n", fileName)
				catStart := time.Now()
				catCount := 0

				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Candidate, error) {
					return extractor.ParseTSECandidate(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertCandidates(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo candidatos")
						} else {
							totalProcessed += len(batch)
							catCount += len(batch)
							printProgress("Candidatos", catCount, catStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()
				printDone("Candidatos", catCount, catStart)
				fmt.Println()

			} else if strings.Contains(fileName, "bem_candidato") {
				fmt.Printf("💎 [Patrimônio & Bens Declarados] Processando %s...\n", fileName)
				catStart := time.Now()
				catCount := 0

				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.CandidateAsset, error) {
					return extractor.ParseTSEAsset(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertAssets(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo bens de candidatos")
						} else {
							totalProcessed += len(batch)
							catCount += len(batch)
							printProgress("Bens Declarados", catCount, catStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()
				printDone("Bens Declarados", catCount, catStart)
				fmt.Println()

			} else if strings.Contains(fileName, "receitas_despesas") || strings.Contains(fileName, "prestacao_de_contas") {
				// Process BOTH Donations AND Expenses from this master package
				// 1. Ingest Donations
				fmt.Printf("💰 [Doações de Campanha] Processando receitas em %s...\n", fileName)
				donStart := time.Now()
				donCount := 0

				poolDon := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Donation, error) {
					return extractor.ParseTSEDonation(row, tseYear)
				})

				batchChanDon := poolDon.Start(ctx)
				var writerWGDon sync.WaitGroup
				writerWGDon.Add(1)

				go func() {
					defer writerWGDon.Done()
					for batch := range batchChanDon {
						if err := svc.BatchInsertDonations(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo doações")
						} else {
							totalProcessed += len(batch)
							donCount += len(batch)
							printProgress("Doações", donCount, donStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSVFilter(ctx, filePath, ';', true, func(entryName string) bool {
					return strings.HasPrefix(entryName, "receitas_candidatos_") && !strings.Contains(entryName, "doador_originario")
				}, func(record []string) error {
					poolDon.Submit(record)
					return nil
				})
				poolDon.Close()
				writerWGDon.Wait()
				printDone("Doações", donCount, donStart)
				fmt.Println()

				// 2. Ingest Expenses
				fmt.Printf("💳 [Despesas & Fornecedores] Processando despesas em %s...\n", fileName)
				expStart := time.Now()
				expCount := 0

				poolExp := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Expense, error) {
					return extractor.ParseTSEExpense(row, tseYear)
				})

				batchChanExp := poolExp.Start(ctx)
				var writerWGExp sync.WaitGroup
				writerWGExp.Add(1)

				go func() {
					defer writerWGExp.Done()
					for batch := range batchChanExp {
						if err := svc.BatchInsertExpenses(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo despesas")
						} else {
							totalProcessed += len(batch)
							expCount += len(batch)
							printProgress("Despesas", expCount, expStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSVFilter(ctx, filePath, ';', true, func(entryName string) bool {
					return strings.HasPrefix(entryName, "despesas_contratadas_candidatos_")
				}, func(record []string) error {
					poolExp.Submit(record)
					return nil
				})
				poolExp.Close()
				writerWGExp.Wait()
				printDone("Despesas", expCount, expStart)
				fmt.Println()

			} else if strings.Contains(fileName, "receitas") {
				fmt.Printf("💰 [Doações de Campanha] Processando %s...\n", fileName)
				catStart := time.Now()
				catCount := 0

				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Donation, error) {
					return extractor.ParseTSEDonation(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertDonations(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo doações")
						} else {
							totalProcessed += len(batch)
							catCount += len(batch)
							printProgress("Doações", catCount, catStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()
				printDone("Doações", catCount, catStart)
				fmt.Println()

			} else if strings.Contains(fileName, "despesas") {
				fmt.Printf("💳 [Despesas & Fornecedores] Processando %s...\n", fileName)
				catStart := time.Now()
				catCount := 0

				pool := worker.NewPool(workersCount, 20000, batchSize, 2*time.Second, func(row []string) (*domain.Expense, error) {
					return extractor.ParseTSEExpense(row, tseYear)
				})

				batchChan := pool.Start(ctx)
				var writerWG sync.WaitGroup
				writerWG.Add(1)

				go func() {
					defer writerWG.Done()
					for batch := range batchChan {
						if err := svc.BatchInsertExpenses(ctx, batch); err != nil {
							log.Error().Err(err).Int("batch_size", len(batch)).Msg("Erro inserindo despesas")
						} else {
							totalProcessed += len(batch)
							catCount += len(batch)
							printProgress("Despesas", catCount, catStart)
						}
					}
				}()

				_ = streamutil.StreamZipCSV(ctx, filePath, ';', true, func(record []string) error {
					pool.Submit(record)
					return nil
				})
				pool.Close()
				writerWG.Wait()
				printDone("Despesas", catCount, catStart)
				fmt.Println()
			}
		}

		fmt.Println("==============================================================================")
		fmt.Printf("🎉 INGESTÃO TOTAL CONCLUÍDA: %s registros gravados no grafo em %s!\n", formatNumber(totalProcessed), time.Since(start).Round(time.Second))
		fmt.Println("==============================================================================")

		return nil
	},
}

func init() {
	ingestTSECmd.Flags().IntVarP(&tseYear, "year", "y", 2022, "Ano da eleição a processar")
	ingestTSECmd.Flags().StringVarP(&sourcePath, "source", "s", "", "Caminho do arquivo ZIP ou diretório do TSE")
	ingestTSECmd.Flags().IntVar(&batchSize, "batch-size", 5000, "Tamanho do lote para gravação no grafo")
	ingestTSECmd.Flags().IntVar(&workersCount, "workers", 8, "Número de workers concorrentes")
	_ = ingestTSECmd.MarkFlagRequired("source")

	ingestCmd.AddCommand(ingestTSECmd)
	rootCmd.AddCommand(ingestCmd)
}
