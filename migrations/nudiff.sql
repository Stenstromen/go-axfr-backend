/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
 SET NAMES utf8mb4 ;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

DROP TABLE IF EXISTS `dates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `dates` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `date` int(11) DEFAULT NULL,
  `amount` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `dateamountindex` (`date`,`amount`),
  KEY `date_idx` (`date`)
) ENGINE=InnoDB AUTO_INCREMENT=972 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `dates` WRITE;
/*!40000 ALTER TABLE `dates` DISABLE KEYS */;
INSERT INTO `dates` (`id`, `date`, `amount`) VALUES (971,20250314,44);
/*!40000 ALTER TABLE `dates` ENABLE KEYS */;
UNLOCK TABLES;

DROP TABLE IF EXISTS `domains`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE `domains` (
  `dategrp` int(11) DEFAULT NULL,
  `domain` varchar(255) DEFAULT NULL,
  KEY `dategrpdomainindex` (`dategrp`,`domain`),
  KEY `domain_idx` (`domain`),
  KEY `dategrp_idx` (`dategrp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `domains` WRITE;
/*!40000 ALTER TABLE `domains` DISABLE KEYS */;
INSERT INTO `domains` (`dategrp`, `domain`) VALUES (971,'badrumonline.nu'),(971,'badrumsdeal.nu'),(971,'badrumsproffs.nu'),(971,'bedrijfsenergievergelijker.nu'),(971,'capisco.nu'),(971,'danser.nu'),(971,'digitalisering.nu'),(971,'elkem.nu'),(971,'fx8.nu'),(971,'gillabyran.nu'),(971,'gocapisco.nu'),(971,'hazen.nu'),(971,'hhab.nu'),(971,'intagsservice.nu'),(971,'jii.nu'),(971,'jonah.nu'),(971,'jouwadvocaat.nu'),(971,'legendslive.nu'),(971,'lunchdags.nu'),(971,'marklinder.nu'),(971,'mdsab.nu'),(971,'movemind.nu'),(971,'oemaayah.nu'),(971,'profixsverige.nu'),(971,'projektera.nu'),(971,'promeet.nu'),(971,'protreptik.nu'),(971,'rum13.nu'),(971,'scouting.nu'),(971,'sinme.nu'),(971,'slackline.nu'),(971,'snall.nu'),(971,'stroomvoorbedrijven.nu'),(971,'swecanab.nu'),(971,'swedshop.nu'),(971,'tagrensning.nu'),(971,'texas-holdem.nu'),(971,'vetterberg.nu'),(971,'vkproffsen.nu'),(971,'werkenmettrauma.nu'),(971,'xn--allamssor-z2a.nu'),(971,'xn--gillabyrn-d3a.nu'),(971,'xn--hemstd-fua.nu'),(971,'zakrisson.nu');
/*!40000 ALTER TABLE `domains` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;
