# Documentação do Backend - TFT Core

## Visão Geral

API REST em Go para consulta de dados do Teamfight Tactics, utilizando cache Redis, PostgreSQL para persistência e NATS para processamento assíncrono.

## Arquitetura

### Componentes Principais

- **API REST**: Endpoints para consulta de jogadores e rankings
- **Cache Redis**: Cache distribuído para otimização de performance
- **PostgreSQL**: Persistência de dados de jogadores
- **NATS**: Sistema de mensageria para processamento assíncrono
- **Rate Limiter**: Controle de taxa baseado em Redis

### Estrutura de Diretórios

```
tft-core/
├── cmd/main.go              # Ponto de entrada da aplicação
├── internal/                # Lógica de negócio
│   ├── config.go           # Configurações
│   ├── models.go           # Estruturas de dados
│   ├── handlers.go         # Handlers HTTP
│   ├── riotapi.go          # Cliente da API Riot
│   ├── cache.go            # Gerenciador de cache
│   ├── database.go         # Gerenciador de banco
│   ├── nats.go             # Cliente NATS
│   └── ratelimiter.go      # Rate limiter
```

## Endpoints

### Saúde
- `GET /healthz` - Status da aplicação

### Jogadores
- `GET /summoner?puuid={puuid}` - Dados do jogador por PUUID
- `GET /search/player?gameName={name}&tagLine={tag}` - Busca jogador por nome
- `GET /league/by-puuid?puuid={puuid}` - Liga do jogador

### Rankings
- `GET /league/challenger` - Top 10 Challenger
- `GET /league/grandmaster` - Top 10 Grandmaster
- `GET /league/master` - Top 10 Master
- `GET /league/entries?tier={tier}&division={div}&page={n}` - Entradas paginadas

## Configuração

### Variáveis de Ambiente

```bash
# Riot API
RIOT_API_KEY=<chave_da_riot>
RIOT_BASE_URL=<url_base_riot>
RIOT_REGION=BR1

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=<usuario>
POSTGRES_PASSWORD=<senha>
POSTGRES_DB=<database>

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=<senha>

# NATS
NATS_URL=nats://localhost:4222

# Aplicação
APP_PORT=8000
CACHE_ENABLED=true
DATABASE_ENABLED=true
```

## Cache Strategy

### TTL por Tipo
- **Summoner Data**: 1 hora
- **Account Data**: 6 horas
- **League Rankings**: 30 minutos
- **Summoner Names**: 24 horas

### Fallback
Redis → PostgreSQL → API Riot → Cache

## Workers Assíncronos

### Summoner Name Worker
- **Tópico**: `tft.summoner.name.fetch`
- **Função**: Enriquece entradas com nomes de jogadores
- **Trigger**: Quando nome não está em cache

### League Update Worker
- **Tópico**: `tft.league.update`
- **Função**: Atualiza rankings em background
- **Frequência**: A cada 30 minutos

## Rate Limiting

### Limites Riot API
- 20 requests/segundo
- 100 requests/2 minutos

### Implementação
- Baseado em Redis com sliding window
- Chaves por endpoint

## Performance

### Otimizações
- Cache multinível (Redis + PostgreSQL)
- Top 10 por tier (limitação de resultados)
- Processamento assíncrono de nomes
- Rate limiting distribuído

### Monitoramento
- Health check com status de serviços
- Logs estruturados
- Timeouts configurados (10s)

### Dependências
- PostgreSQL 17.0
- Redis 8.0
- NATS Server 2.11.7

## Observabilidade

### Logs
- Structured logging
- Connection status
- Worker processing
- Error tracking

### Métricas
- Request/response timing
- Cache hit/miss rates
- Worker queue depth
- API error rates

## Database Schema

### Tabela: summoner_cache

```sql
CREATE TABLE summoner_cache (
    puuid VARCHAR(78) PRIMARY KEY,
    game_name VARCHAR(32) NOT NULL,
    tag_line VARCHAR(8) NOT NULL,
    summoner_id VARCHAR(63),
    region VARCHAR(8) NOT NULL DEFAULT 'BR1',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Índices
- `idx_summoner_cache_name`: (game_name, tag_line)
- `idx_summoner_cache_region`: (region)
- `idx_summoner_cache_updated`: (last_updated)

## Troubleshooting

### Health Check Response

```json
{
  "status": "ok",
  "timestamp": 1234567890,
  "services": {
    "redis": "connected",
    "nats": "connected"
  }
}
```